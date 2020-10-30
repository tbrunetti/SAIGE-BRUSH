package main

import (
	"fmt"
	"os/exec"
	"sync"
	"strings"
	"strconv"
	"path/filepath"
	"os"
	"bufio"
	"io"
	"math"
	"runtime"
	"time"
)

var wgChunk sync.WaitGroup
var chrLengths = make(map[string]int)
var processQueue = make([]string, 0)
var allChunksFinished int
var allAssocationsRunning int
var wgPipeline sync.WaitGroup
var wgSmallChunk sync.WaitGroup
var wgNull sync.WaitGroup
var wgAssociation sync.WaitGroup
var wgAllChunks sync.WaitGroup
var changeQueueSize sync.Mutex
var queueCheck sync.Mutex
var errorHandling sync.Mutex



func main() {
	runtime.GOMAXPROCS(24)
	nprocs := 24

	allChunksFinished = 1
	// variables that need to be acquired by parsing through user input
	chunkVariants := 2000000
	chromosomeLengthFile := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/hg38_chrom_sizes.txt"
	build := "hg38"
	chromosomes := "1-22"
	imputeSuffix := "_rsq70_merged_renamed.vcf.gz"
	imputeDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/TOPMedImputation"
	bindPoint := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/"
	container := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/SAIGE_v0.39_CCPM_biobank_singularity_recipe_file_10272020.simg"
	outDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/test_new_pipeline/"
	outPrefix := "GO_TEST_multiple_sclerosis_CCPMbb_freeze_v1.3"
	sparseGRM := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/step0_GRM/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned_relatednessCutoff_0.0625_103154_randomMarkersUsed.sparseGRM.mtx"
	sampleIDFile := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/step0_GRM/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned_relatednessCutoff_0.0625_103154_randomMarkersUsed.sparseGRM.mtx.sampleIDs.txt"
	phenoFile := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/biobank_paper_pheWAS/pheWAS_CCPMbb_freeze_v1.3.txt" 
	plink := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/LDprunedMEGA/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned"
	trait := "binary"
	pheno := "multiple_sclerosis"
	invNorm := "FALSE"
	covars := "PC1,PC2,PC3,PC4,PC5,SAIGE_GENDER,age"
	sampleID := "FULL_BBID"
	nThreads := "10"
	sparseKin := "TRUE"
	markers := "30"
	rel := "0.0625"
	loco := "TRUE"
	covTransform := "TRUE"
	vcfField := "DS" // DS or GT
	MAF := "0.5"
	MAC := "10"
	IsDropMissingDosages := "FALSE"
	infoFile := "allAutosomes.rs70.info.SAIGE.txt"

	// end of variables
	

 	chroms:= strings.Split(chromosomes, "-")
 	start:= strings.TrimSpace(chroms[0])
 	end:= strings.TrimSpace(chroms[1])

 	wgNull.Add(1)
	go nullModel(bindPoint,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outDir,outPrefix,rel,loco,covTransform) 

 	// chunk can run the same time as the null model; and the association analysis needs to wait for the null to finish and chunk to finish
  	wgAllChunks.Add(1)
    go chunk(start,end,build,outDir,chromosomeLengthFile,imputeDir,imputeSuffix,bindPoint,container,chunkVariants)
    
    wgNull.Wait()

	// while loop to keep submitting jobs until queue is empty and no more subsets are available
	for allChunksFinished == 1 || len(processQueue) != 0 {
		if allAssocationsRunning < (nprocs*2) && len(processQueue) > 0 {
			changeQueueSize.Lock()
			vcfFile := processQueue[0]
			tmp := strings.Split(vcfFile, "_")
			subName := strings.TrimSuffix(vcfFile, imputeSuffix)
			if tmp[0]+imputeSuffix == vcfFile {
				wgAssociation.Add(1)
				go associationAnalysis(bindPoint,container,filepath.Join(imputeDir,vcfFile),vcfField,outDir,tmp[0],subName,sampleIDFile,IsDropMissingDosages,outPrefix,loco)
				processQueue = processQueue[1:]
				time.Sleep(2* time.Second)
			}else{
				wgAssociation.Add(1)
				go associationAnalysis(bindPoint,container,filepath.Join(outDir,vcfFile),vcfField,outDir,tmp[0],subName,sampleIDFile,IsDropMissingDosages,outPrefix,loco)
				processQueue = processQueue[1:]
				time.Sleep(2* time.Second)
			}
			changeQueueSize.Unlock()
		}else{
			time.Sleep(5* time.Minute)
		}
	}

    wgAssociation.Wait()

    concat := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", concat.Year(), concat.Month(), concat.Day(), concat.Hour(), concat.Minute(), concat.Second())
    fmt.Printf("[func(main) %s] Concatenating all association results...\n", formatted)
    concat := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/concatenate.sh", filepath.Join(outDir,outPrefix))
    concat.Run()

    errorHandling.Lock()
    concat.Stdout = os.Stdout
    concat.Stderr = os.Stderr
    errorHandling.Unlock()
    
    fmt.Printf("[func(main) -- concatenate] Finished all association results. Time Elapsed: %.2f minutes\n", time.Since(concat).Minutes())

    
    graph := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", graph.Year(), graph.Month(), graph.Day(), graph.Hour(), graph.Minute(), graph.Second())
    fmt.Printf("[func(main) -- clean and graph results] Start data clean up, visualization, and summarization...\n")
    cleanAndGraph := exec.Command("singularity", "run", "-B", bindPoint, container, "/usr/lib/R/bin/Rscript", "/opt/cleanAndGraph.R",
			"--assocFile="+filepath.Join(outDir,outPrefix) + "_allChromosomeResultsMerged.txt",
			"--infoFile="+infoFile,
			"--dataOutputPrefix="+filepath.Join(outDir,outPrefix),
			"--pheno="+pheno,
			"--covars="+covars,
			"--macFilter="+MAC,
			"--mafFilter="+MAF,
			"--traitType="+trait,
			"--nThreads="+nThreads)
    cleanAndGraph.Run()

    errorHandling.Lock()
    graph.Stdout = os.Stdout
    graph.Stderr = os.Stderr
    errorHandling.Unlock()

    fmt.Printf("[func(main) -- clean and graph results] Finished all data clean up, visualizations, and summarization. Time Elapsed: %.2f minutes\n", time.Since(graph).Minutes())

    fmt.Printf("[func(main)] All threads are finished and pipeline is complete!\n")
}


func chunk(start,end,build,outDir,chromosomeLengthFile,imputeDir,imputeSuffix,bindPoint,container string, chunkVariants int) {
	defer wgAllChunks.Done()

	fileBytes, err := os.Open(chromosomeLengthFile)
	if err != nil {
		fmt.Printf("There was a problems reading in the chromosome length file.")
		os.Exit(42)
	}

	defer fileBytes.Close()

	scanBytes := bufio.NewReader(fileBytes)
	//var line string
	for {
		line, err := scanBytes.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Println("An error occurred when reading in the chromosome length file.")
			os.Exit(42)
		}

		if err == io.EOF {
			fmt.Println("Finished reading chromosome file length")
			break
		}
		
		tmp := strings.Split(line, "\t")
		intVal, _ := strconv.Atoi(strings.TrimSpace(tmp[1]))
		//fmt.Printf("%v\n", intVal)
		chrLengths[tmp[0]] = intVal
	}

	//fmt.Printf("%v", chrLengths)
	startInt,_ :=  strconv.Atoi(start)
	endInt,_ := strconv.Atoi(end)

	for i:=startInt; i<endInt+1; i++ {
		chromInt:= strconv.Itoa(i)
		wgChunk.Add(1)
		go smallerChunk(chromInt,build,outDir,imputeDir,imputeSuffix,bindPoint,container,chunkVariants)
	}
	wgChunk.Wait()

	allChunksFinished = 0

}

func smallerChunk(chrom,build,outDir,imputeDir,imputeSuffix,bindPoint,container string, chunkVariants int){
	defer wgChunk.Done()

	// hg38 requires the chr prefix
	if build == "hg38" {
		chrom = "chr"+chrom
	}

	// calculate full loops (not partial where total variants in loops remainder < chunkVariants which is always the last loop)
	loops := int(math.Floor(float64(chrLengths[chrom])/float64(chunkVariants)))
	maxLoops := loops + 1

	//totalVariants is a byte slice but can be converted to a single string by casting it as a string() -- returns total variants in file
	totalVariants,err := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/bcftools",
		"index",
		"--nrecords",
		filepath.Join(imputeDir, chrom+imputeSuffix)).Output()

	errorHandling.Lock()
	totalVariants.Stdout = os.Stdout
	totalVariants.Stderr = os.Stderr
	errorHandling.Unlock()



	// if error is seen, print error and exit out of function, otherwise print the total variants in the vcf file and continue
	if err != nil {
		tErr := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", tErr.Year(), tErr.Month(), tErr.Day(),tErr.Hour(), tErr.Minute(), tErr.Second())
		fmt.Printf("[func(smallerChunk) %s] %s overall: Error in total variants call:\n%v", formatted,chrom, err)
		return
	} else {
		tErr := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", tErr.Year(), tErr.Month(), tErr.Day(),tErr.Hour(), tErr.Minute(), tErr.Second())
		fmt.Printf("[func(smallerChunk) %s] A total of %s variants are in the vcf file for %s\n", formatted,strings.TrimSpace(string(totalVariants)),chrom)
	}

	// convert byte slice to string, trim any trailing whitespace of either end and then convert to integer
	varVal,err := strconv.Atoi(strings.TrimSpace(string(totalVariants)))
	if err != nil{
		tErr := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", tErr.Year(), tErr.Month(), tErr.Day(),tErr.Hour(), tErr.Minute(), tErr.Second())
		fmt.Printf("[func(smallerChunk) %s] There was an error converting the total variants to an integer, likely due to lack of SNPs in the region. The error encountered is:\n%v\n",formatted,err)
		return
	}
	

	// if chunk is larger than total variants in file, do not chunk and just add full imputation file to queue
	if chunkVariants > 200*varVal {
		changeQueueSize.Lock()
		processQueue = append(processQueue, chrom+imputeSuffix)
		changeQueueSize.Unlock()
		t0 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
		fmt.Printf("[func(smallerChunk) %s] %s is in the queue and %d is variant value\n", formatted,chrom+imputeSuffix,varVal)
	}else {
		for loopId := 1; loopId < maxLoops + 1; loopId++ {
			wgSmallChunk.Add(1)
			go processing(loopId,chunkVariants,bindPoint,container,chrom,outDir,imputeDir,imputeSuffix)
		}
	}
	wgSmallChunk.Wait()

}

func processing (loopId,chunkVariants int, bindPoint,container,chrom,outDir,imputeDir,imputeSuffix string) {
	defer wgSmallChunk.Done()
	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(processing) %s] Processing %s, chunk %d\n", formatted, chrom, loopId)
	
	loopNum := strconv.Itoa(loopId)
	upperVal := loopId*chunkVariants
	lowerVal := (loopId*chunkVariants)-(chunkVariants)+1
	upperValStr := strconv.Itoa(upperVal)
	lowerValStr := strconv.Itoa(lowerVal) 
	subset := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/bcftools",
		"view",
		"--regions",
		chrom+":"+lowerValStr+"-"+upperValStr,
		"-Oz",
		"-o",
		filepath.Join(outDir, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix,
		filepath.Join(imputeDir, chrom+imputeSuffix))
	subset.Run()

	errorHandling.Lock()
    subset.Stdout = os.Stdout
    subset.Stderr = os.Stderr
    errorHandling.Unlock()


	index := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/tabix",
	"-p",
	"vcf",
	filepath.Join(outDir, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)
	index.Run()

	errorHandling.Lock()
   	index.Stdout = os.Stdout
    index.Stderr = os.Stderr
    errorHandling.Unlock()

	totalVariants,_ := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/bcftools",
		"index",
		"--nrecords",
		filepath.Join(outDir, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix).Output()

	errorHandling.Lock()
    totalVariants.Stdout = os.Stdout
    totalVariants.Stderr = os.Stderr
    errorHandling.Unlock()

				
	varVal,err1 := strconv.Atoi(strings.TrimSpace(string(totalVariants)))
	if err1 != nil{
		t0 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
		fmt.Printf("[func(processing) %s] %s, chunk %s:\n\tThere was an error converting the total variants to an integer, likely due to lack of SNPs in the region. The error encountered is:\n%v\n",formatted,chrom,loopNum,err1)
		return
	}
			
	if varVal > 0 {
		changeQueueSize.Lock()
		processQueue = append(processQueue, chrom+"_"+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)
		changeQueueSize.Unlock()
		t1 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
		fmt.Printf("[func(processing) %s] %s, chunk %s has successfully completed and has been added to the processing queue. Time Elapsed: %.2f minutes\n", formatted,chrom,loopNum, time.Since(t0).Minutes())
	}else{
		t1 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
		fmt.Printf("[func(processing) %s] %s is empty with value %d and will not be added to queue.\n", formatted, chrom+"_"+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix, varVal)
	}
}
	

func nullModel (bindPoint,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outDir,outPrefix,rel,loco,covTransform string) {
	defer wgNull.Done()
	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(nullModel) %s] Starting Null Model...\n", formatted)

	cmd := exec.Command("singularity", "run", "-B", bindPoint, container, "/usr/lib/R/bin/Rscript", "/opt/step1_fitNULLGLMM.R",
		"--sparseGRMFile="+sparseGRM,
		"--sparseGRMSampleIDFile="+sampleIDFile,
		"--phenoFile="+phenoFile,
		"--plinkFile="+plink,
		"--traitType="+trait,
		"--phenoCol="+pheno,
		"--invNormalize="+invNorm,
		"--covarColList="+covars,
		"--sampleIDColinphenoFile="+sampleID,
		"--nThreads="+nThreads,
		"--IsSparseKin="+sparseKin,
		"--numRandomMarkerforVarianceRatio="+markers,
		"--skipModelFitting=False",
		"--memoryChunk=5",
		"--outputPrefix="+filepath.Join(outDir, outPrefix),
		"--relatednessCutoff="+rel,
		"--LOCO="+loco,
		"--isCovariateTransform="+covTransform)

	cmd.Run()
	
	errorHandling.Lock()
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    errorHandling.Unlock()

	t1 := time.Now()
	formatted = fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
	fmt.Printf("[func(nullModel) %s] Finished Null Model. Time Elapsed: %.2f minutes\n", formatted, time.Since(t0).Minutes())
}


func associationAnalysis(bindpoint,container,vcfFile,vcfField,outDir,chrom,subName,sampleIDFile,IsDropMissingDosages,outPrefix,loco string) {
	defer wgAssociation.Done()
	queueCheck.Lock()
	allAssocationsRunning++
	queueCheck.Unlock()

	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(associationAnalysis) %s] Starting Association of %s...\n", formatted, vcfFile)

		
	cmd := exec.Command("singularity", "run", "-B", bindpoint, container, "/usr/lib/R/bin/Rscript", "/opt/step2_SPAtests.R",
		"--vcfFile="+vcfFile,
		"--vcfFileIndex="+vcfFile+".tbi",
		"--vcfField="+vcfField,
		"--sampleFile="+sampleIDFile,
		"--chrom="+chrom,
		"--IsDropMissingDosages="+IsDropMissingDosages,
		"--minMAF=0",
		"--minMAC=0",
		"--GMMATmodelFile="+filepath.Join(outDir, outPrefix)+".rda",
		"--varianceRatioFile="+filepath.Join(outDir, outPrefix)+".varianceRatio.txt",
		"--numLinesOutput=2",
		"--IsOutputAFinCaseCtrl=TRUE",
		"--IsOutputHetHomCountsinCaseCtrl=TRUE",
		"--IsOutputBETASEinBurdenTest=TRUE",
		"--SAIGEOutputFile="+filepath.Join(outDir, outPrefix)+"_"+chrom+"_SNPassociationAnalysis.txt",
		"--LOCO="+loco)
	cmd.Run()
	
	errorHandling.Lock()
    cmd.Stdout = os.Stdout
    cmdts.Stderr = os.Stderr
    errorHandling.Unlock()

	t1 := time.Now()
	formatted = fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
	fmt.Printf("[func(associationAnalysis) %s] %s has successfully completed. Time Elapsed: %.2f minutes\n", formatted,vcfFile,time.Since(t0).Minutes())

	queueCheck.Lock()
	allAssocationsRunning--
	queueCheck.Unlock()
}