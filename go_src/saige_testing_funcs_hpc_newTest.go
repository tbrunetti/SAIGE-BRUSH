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

var queueFileSave sync.Mutex
var chrLengths = make(map[string]int)
var processQueue = make([]string, 0)
var allChunksFinished int
var allAssocationsRunning int
var totalChunksProcessing int
var wgChunk sync.WaitGroup
var wgPipeline sync.WaitGroup
var wgSmallChunk sync.WaitGroup
var wgNull sync.WaitGroup
var wgAssociation sync.WaitGroup
var wgAllChunks sync.WaitGroup
var changeQueueSize sync.Mutex
var queueCheck sync.Mutex
var errorHandling sync.Mutex
var chunkQueue sync.Mutex


//TODO: PRINT OUT QUEUE DOC FOR THOSE THAT WANT TO REUSE CHUNKS -- implemented and needs testing 
//TODO: Add Option to use queue doc and implement -- implemented needs testing
//TODO: ALSO ADD OPTION TO SAVE CHUNKS for parsing -- logic within code and what to do with flag is already implemented
//TODO: confirm all path and columns exist from user else, throw error
//TODO: Still need to print out stdout and errors from exec.Command() calls
//TODO: In cleanAndGraph make sure dataframes are not empty before graphing or skip graph for that data set
//TODO: clean up temp files  -- implemented but needs testing
//TODO: use totalCPUsAvail to determine how many threads to allocate to Null (if no chunking required use all selse use certain percentage -- use full for graphing)
//TODO: if sparse kinship is not provided, automatically generate one
//TODO: for quant traits default for --invNormalize=" in null model should be set to true ,for binary default is false when parsing user input

func main() {
	totalCPUsAvail := runtime.NumCPU()
	runtime.GOMAXPROCS(totalCPUsAvail)
	fmt.Printf("%v total CPUs available.\n", totalCPUsAvail)

	allChunksFinished = 1
	// variables that need to be acquired by parsing through user input
	chunkVariants := 1000000
	chromosomeLengthFile := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/hg38_chrom_sizes.txt"
	build := "hg38"
	chromosomes := "21-22"
	imputeSuffix := "_rsq70_merged_renamed.vcf.gz"
	//imputeDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/TOPMedImputation"
	imputeDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/test_new_pipeline"
	bindPoint := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/"
	bindPointTemp := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/tmp/"
	container := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/SAIGE_v0.39_CCPM_biobank_singularity_recipe_file_11092020.simg"
	//outDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/test_new_pipeline/"
	outDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/test_new_pipeline_11122020"
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
	nThreads := "24"
	sparseKin := "TRUE"
	markers := "30"
	rel := "0.0625"
	loco := "TRUE"
	covTransform := "TRUE"
	vcfField := "DS" // DS or GT
	MAF := "0.05"
	MAC := "10"
	IsDropMissingDosages := "FALSE"
	infoFile := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/TOPMedImputationInfo/allAutosomes.rsq70.info.SAIGE.txt"
	saveChunks := false
	imputationFileList := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/test_new_pipeline/GO_TEST_multiple_sclerosis_CCPMbb_freeze_v1.3_chunkedImputationQueue.txt" // this is required if skipChunking is set to true
	skipChunking := true
	// end of variables

	// split chroms
 	chroms:= strings.Split(chromosomes, "-")
 	start:= strings.TrimSpace(chroms[0])
 	end:= strings.TrimSpace(chroms[1])

 	f,_:= os.Create(filepath.Join(bindPointTemp, outPrefix+"_chunkedImputationQueue.txt"))
	defer f.Close()

 	// STEP1: Run Null Model Queue
 	wgNull.Add(1)
	go nullModel(bindPoint,bindPointTemp,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outDir,outPrefix,rel,loco,covTransform) 

 	
 	/* STEP1a: Chunks Queue or Use of previously chunked imputation data
 	chunk can run the same time as the null model; and the association analysis needs to wait for the 
 	null to finish and chunk to finish*/
 	if skipChunking ==  false {
 		wgAllChunks.Add(1)
    	go chunk(start,end,build,outDir,chromosomeLengthFile,imputeDir,imputeSuffix,bindPoint,bindPointTemp,container,chunkVariants,f)
 	}else {
 		wgAllChunks.Add(1)
 		go usePrevChunks(imputeDir, imputationFileList)
 	}
  
    
    // wait for null model to finish before proceeding with association analysis -- no need to wait for chunk to finish
    wgNull.Wait()

	
    /* STEP2: Association Analysis Queue
	while loop to keep submitting jobs until queue is empty and no more subsets are available*/
	for allChunksFinished == 1 || len(processQueue) != 0 {
		if allAssocationsRunning < (totalCPUsAvail*2) && len(processQueue) > 0 {
			changeQueueSize.Lock() // lock queue to prevent collision
			vcfFile := processQueue[0]
			tmp := strings.Split(vcfFile, "_") // extract chromosome name
			subName := strings.TrimSuffix(vcfFile, imputeSuffix)
			// if condition is true, that means the data is not chunked and need to be read from the imputeDir not outDir
			if ((tmp[0]+imputeSuffix == vcfFile) || (skipChunking == true)) {
				wgAssociation.Add(1)
				go associationAnalysis(bindPoint,bindPointTemp,container,filepath.Join(imputeDir,vcfFile),vcfField,outDir,tmp[0],subName,sampleIDFile,IsDropMissingDosages,outPrefix,loco)
				processQueue = processQueue[1:]
				time.Sleep(1* time.Second) // prevent queue overload
			}else{
				wgAssociation.Add(1)
				go associationAnalysis(bindPoint,bindPointTemp,container,filepath.Join(bindPointTemp,vcfFile),vcfField,outDir,tmp[0],subName,sampleIDFile,IsDropMissingDosages,outPrefix,loco)
				processQueue = processQueue[1:]
				time.Sleep(1* time.Second) // prevent queue overload
			}
			changeQueueSize.Unlock() // unlock queue safely
		}else{
			time.Sleep(5* time.Minute)
		}
	}

	// wait for all association anlayses to finish before proceeding
    wgAssociation.Wait()

    // STEP2a: Concatenate all results into one file
    concatTime := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", concatTime.Year(), concatTime.Month(), concatTime.Day(), concatTime.Hour(), concatTime.Minute(), concatTime.Second())
    fmt.Printf("[func(main) %s] Concatenating all association results...\n", formatted)
    concat := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/concatenate.sh", filepath.Join(bindPointTemp,outPrefix))
    concat.Stdout = os.Stdout
    concat.Stderr = os.Stderr
    concat.Run()
    
    fmt.Printf("[func(main) -- concatenate] Finished all association results. Time Elapsed: %.2f minutes\n", time.Since(concatTime).Minutes())

    // STEP3: Clean, Visualize, and Summarize results
    graph := time.Now()
	formatted = fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", graph.Year(), graph.Month(), graph.Day(), graph.Hour(), graph.Minute(), graph.Second())
    fmt.Printf("[func(main) -- clean and graph results] Start data clean up, visualization, and summarization...\n")
    cleanAndGraph := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/usr/lib/R/bin/Rscript", "/opt/step3_GWASsummary.R",
			"--assocFile="+filepath.Join(bindPointTemp, outPrefix) + "_allChromosomeResultsMerged.txt",
			"--infoFile="+infoFile,
			"--dataOutputPrefix="+filepath.Join(bindPointTemp, outPrefix),
			"--pheno="+pheno,
			"--covars="+covars,
			"--macFilter="+MAC,
			"--mafFilter="+MAF,
			"--traitType="+trait,
			"--nThreads="+nThreads)
	cleanAndGraph.Stdout = os.Stdout
    cleanAndGraph.Stderr = os.Stderr
    cleanAndGraph.Run()


    fmt.Printf("[func(main) -- clean and graph results] Finished all data clean up, visualizations, and summarization. Time Elapsed: %.2f minutes\n", time.Since(graph).Minutes())

    //TODO: clean up temp files
    saveResults(bindPointTemp,outDir,saveChunks)

    // Pipeline Finish
    fmt.Printf("[func(main)] All threads are finished and pipeline is complete!\n")
}




func chunk(start,end,build,outDir,chromosomeLengthFile,imputeDir,imputeSuffix,bindPoint,bindPointTemp,container string, chunkVariants int, f *os.File) {
	defer wgAllChunks.Done() //once function finishes decrement sync object

	fileBytes, err := os.Open(chromosomeLengthFile)
	if err != nil {
		fmt.Printf("There was a problems reading in the chromosome length file.")
		os.Exit(42)
	}

	defer fileBytes.Close() // once funciton finished close file

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
		chrLengths[tmp[0]] = intVal
	}

	startInt,_ :=  strconv.Atoi(start)
	endInt,_ := strconv.Atoi(end)

	
	for i:=startInt; i<endInt+1; i++ {
		chromInt:= strconv.Itoa(i)
		wgChunk.Add(1)
		smallerChunk(chromInt,build,outDir,imputeDir,imputeSuffix,bindPoint,bindPointTemp,container,chunkVariants,f)
		time.Sleep(1* time.Second)
	}

	allChunksFinished = 0

}

func smallerChunk(chrom,build,outDir,imputeDir,imputeSuffix,bindPoint,bindPointTemp,container string, chunkVariants int, f *os.File){
	defer wgChunk.Done()

	// hg38 requires the chr prefix
	if build == "hg38" {
		chrom = "chr"+chrom
	}

	// calculate full loops (not partial where total variants in loops remainder < chunkVariants which is always the last loop)
	loops := int(math.Floor(float64(chrLengths[chrom])/float64(chunkVariants)))
	maxLoops := loops + 1

	//totalVariants is a byte slice but can be converted to a single string by casting it as a string() -- returns total variants in file
	totalVariants,err := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/bcftools",
		"index",
		"--nrecords",
		filepath.Join(imputeDir, chrom+imputeSuffix)).Output()


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
	varVal,_ := strconv.Atoi(strings.TrimSpace(string(totalVariants)))
	/*if err != nil{
		tErr := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", tErr.Year(), tErr.Month(), tErr.Day(),tErr.Hour(), tErr.Minute(), tErr.Second())
		fmt.Printf("[func(smallerChunk) %s] There was an error converting the total variants to an integer, likely due to lack of SNPs in the region. The error encountered is:\n%v\n",formatted,err)
		return
	}*/
	

	// if chunk is larger than total variants in file, do not chunk and just add full imputation file to queue
	if chunkVariants > 200*varVal {
		changeQueueSize.Lock() // lock access to shared queue to prevent collision
		processQueue = append(processQueue, chrom+imputeSuffix)
		changeQueueSize.Unlock() // unlock access to shared queue
		saveQueue(chrom+imputeSuffix,f)
		t0 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
		fmt.Printf("[func(smallerChunk) %s] %s is in the queue and %d is variant value\n", formatted,chrom+imputeSuffix,varVal)
	}else {
		for loopId := 1; loopId < maxLoops + 1; loopId++ {
			wgSmallChunk.Add(1)
			go processing(loopId,chunkVariants,bindPoint,bindPointTemp,container,chrom,outDir,imputeDir,imputeSuffix,f)
			time.Sleep(1* time.Second)
		}
	}
	wgSmallChunk.Wait()

}

func processing (loopId,chunkVariants int, bindPoint,bindPointTemp,container,chrom,outDir,imputeDir,imputeSuffix string, f *os.File) {
	defer wgSmallChunk.Done()
	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(processing) %s] Processing %s, chunk %d\n", formatted, chrom, loopId)
	
	loopNum := strconv.Itoa(loopId)
	upperVal := loopId*chunkVariants
	lowerVal := (loopId*chunkVariants)-(chunkVariants)+1
	upperValStr := strconv.Itoa(upperVal)
	lowerValStr := strconv.Itoa(lowerVal) 
	subset := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/bcftools",
		"view",
		"--regions",
		chrom+":"+lowerValStr+"-"+upperValStr,
		"-Oz",
		"-o",
		filepath.Join(bindPointTemp, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix,
		filepath.Join(imputeDir, chrom+imputeSuffix))
	subset.Run()

	errorHandling.Lock()
    subset.Stdout = os.Stdout
    subset.Stderr = os.Stderr
    errorHandling.Unlock()


	index := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/tabix",
		"-p",
		"vcf",
		filepath.Join(bindPointTemp, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)
	index.Run()

	errorHandling.Lock()
   	index.Stdout = os.Stdout
    index.Stderr = os.Stderr
    errorHandling.Unlock()

	totalVariants,_ := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/bcftools",
		"index",
		"--nrecords",
		filepath.Join(bindPointTemp, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix).Output()

	fmt.Printf("%v, %s chunk %s, %s-%s", strings.TrimSpace(string(totalVariants)), chrom, loopNum, lowerValStr, upperValStr)				
	varVal,_ := strconv.Atoi(strings.TrimSpace(string(totalVariants)))
	/*if err1 != nil{
		t0 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
		fmt.Printf("[func(processing) %s] %s, chunk %s:\n\tThere was an error converting the total variants to an integer, likely due to lack of SNPs in the region. The error encountered is:\n%v\n",formatted,chrom,loopNum,err1)
		return
	}*/
			
	if varVal > 0 {
		changeQueueSize.Lock()
		processQueue = append(processQueue, chrom+"_"+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)
		changeQueueSize.Unlock()
		saveQueue(chrom+"_"+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix,f)
		t1 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
		fmt.Printf("[func(processing) %s] %s, chunk %s has successfully completed and has been added to the processing queue. Time Elapsed: %.2f minutes\n", formatted,chrom,loopNum, time.Since(t0).Minutes())
	}else{
		t1 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
		fmt.Printf("[func(processing) %s] %s is empty with value %d and will not be added to queue.\n", formatted, chrom+"_"+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix, varVal)
	}
}
	

func nullModel (bindPoint,bindPointTemp,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outDir,outPrefix,rel,loco,covTransform string) {
	defer wgNull.Done() // decrement wgNull sync object
	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(nullModel) %s] Starting Null Model...\n", formatted)

	switch{
	case strings.ToLower(strings.TrimSpace(trait)) == "binary":
		
		// singularity command to run step1: null model for true binary (0 controls, 1 cases) traits
		cmd := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/usr/lib/R/bin/Rscript", "/opt/step1_fitNULLGLMM.R",
			"--sparseGRMFile="+sparseGRM,
			"--sparseGRMSampleIDFile="+sampleIDFile,
			"--phenoFile="+phenoFile,
			"--plinkFile="+plink,
			"--traitType=binary",
			"--phenoCol="+pheno,
			"--invNormalize="+invNorm,
			"--covarColList="+covars,
			"--sampleIDColinphenoFile="+sampleID,
			"--nThreads="+nThreads,
			"--IsSparseKin="+sparseKin,
			"--numRandomMarkerforVarianceRatio="+markers,
			"--skipModelFitting=False",
			"--memoryChunk=5",
			"--outputPrefix="+filepath.Join(bindPointTemp, outPrefix),
			"--relatednessCutoff="+rel,
			"--LOCO="+loco,
			"--isCovariateTransform="+covTransform)

		cmd.Run() // run automatically wait for null to finish before processing next lines within function
	
		errorHandling.Lock()
    	cmd.Stdout = os.Stdout
    	cmd.Stderr = os.Stderr
    	errorHandling.Unlock()

	case strings.ToLower(strings.TrimSpace(trait)) == "quantitative":
		// singularity command to run step1: null model for quantitative traits
		cmd := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/usr/lib/R/bin/Rscript", "/opt/step1_fitNULLGLMM.R",
			"--sparseGRMFile="+sparseGRM,
			"--sparseGRMSampleIDFile="+sampleIDFile,
			"--phenoFile="+phenoFile,
			"--plinkFile="+plink,
			"--traitType=quantitative",
			"--phenoCol="+pheno,
			"--invNormalize="+invNorm,
			"--covarColList="+covars,
			"--sampleIDColinphenoFile="+sampleID,
			"--nThreads="+nThreads,
			"--IsSparseKin="+sparseKin,
			"--numRandomMarkerforVarianceRatio="+markers,
			"--skipModelFitting=False",
			"--memoryChunk=5",
			"--outputPrefix="+filepath.Join(bindPointTemp, outPrefix),
			"--relatednessCutoff="+rel,
			"--LOCO="+loco,
			"--isCovariateTransform="+covTransform,
			"--tauInit=1,0")

		cmd.Run() // run automatically wait for null to finish before processing next lines within function
	
		errorHandling.Lock()
    	cmd.Stdout = os.Stdout
    	cmd.Stderr = os.Stderr
    	errorHandling.Unlock()	

	default:
		fmt.Printf("Please select trait type as either binary or quantitative.  You entered: %s.\n", trait)
		os.Exit(42)
	}

	t1 := time.Now()
	formatted = fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
	fmt.Printf("[func(nullModel) %s] Finished Null Model. Time Elapsed: %.2f minutes\n", formatted, time.Since(t0).Minutes())
}


func associationAnalysis(bindpoint,bindPointTemp,container,vcfFile,vcfField,outDir,chrom,subName,sampleIDFile,IsDropMissingDosages,outPrefix,loco string) {
	defer wgAssociation.Done() // decrement wgAssociation sync object when function finishes
	queueCheck.Lock() // lock the number of associations running to prevent collision
	allAssocationsRunning++
	queueCheck.Unlock() // unlock shared variable safely

	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(associationAnalysis) %s] Starting Association of %s...\n", formatted, vcfFile)

	// run step2: association analyses command in SAIGE via Singularity		
	cmd := exec.Command("singularity", "run", "-B", bindpoint+","+bindPointTemp, container, "/usr/lib/R/bin/Rscript", "/opt/step2_SPAtests.R",
		"--vcfFile="+vcfFile,
		"--vcfFileIndex="+vcfFile+".tbi",
		"--vcfField="+vcfField,
		"--sampleFile="+sampleIDFile,
		"--chrom="+chrom,
		"--IsDropMissingDosages="+IsDropMissingDosages,
		"--minMAF=0",
		"--minMAC=0",
		"--GMMATmodelFile="+filepath.Join(bindPointTemp, outPrefix)+".rda",
		"--varianceRatioFile="+filepath.Join(bindPointTemp, outPrefix)+".varianceRatio.txt",
		"--numLinesOutput=2",
		"--IsOutputAFinCaseCtrl=TRUE",
		"--IsOutputHetHomCountsinCaseCtrl=TRUE",
		"--IsOutputBETASEinBurdenTest=TRUE",
		"--IsOutputNinCaseCtrl=TRUE",
		"--SAIGEOutputFile="+filepath.Join(bindPointTemp, outPrefix)+"_"+subName+"_SNPassociationAnalysis.txt",
		"--LOCO="+loco)
	cmd.Run()
	
	errorHandling.Lock()
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    errorHandling.Unlock()

	t1 := time.Now()
	formatted = fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
	fmt.Printf("[func(associationAnalysis) %s] %s has successfully completed. Time Elapsed: %.2f minutes\n", formatted,vcfFile,time.Since(t0).Minutes())

	queueCheck.Lock() // lock shared variable to prevent collision
	allAssocationsRunning--
	queueCheck.Unlock() // unlock access to shared variable
}

func checkInput(MAC,MAF string) {

	// check MAC
	checkMAC,err := strconv.ParseFloat(MAF, 64)
	if err != nil {
		fmt.Printf("[func(checkInput)] There was an error converting MAC to float 64. See following : %v\n", err)
		os.Exit(42)
	} else if checkMAC < 0.0 {
		fmt.Printf("[func(checkInput)] Error: minor allele count cannot be smaller (negative) than 0 (0%). Please select a positive value.\n")
		os.Exit(42)
	}

	// check MAF
	checkMAF,err := strconv.ParseFloat(MAF, 64)
	if err != nil {
		fmt.Printf("[func(checkInput)] There was an error converting MAF to float 64. See following : %v\n", err)
		os.Exit(42)
	} else if checkMAF > 0.50 {
		fmt.Printf("[func(checkInput)] Error: minor allele frequency cannot be larger than 0.50 (50%). Please select a value between 0.0-0.50.\n")
		os.Exit(42)
	} else if checkMAF < 0.0 {
		fmt.Printf("[func(checkInput)] Error: minor allele frequency cannot be smaller (negative) than 0 (0%). Please select a value between 0.0-0.50.\n")
		os.Exit(42)
	}
}

func saveResults(bindPointTemp,outDir string, saveChunks bool) {
	save := time.Now()
	fmt.Printf("[func(saveResults) -- begin transferring final results]\n")
	matches := make([]string, 0)
    findThese := [12]string{"*.mtx.sampleIDs.txt", "*.sparseGRM.mtx", "*.sparseSigma.mtx", 
    			"*.varianceRatio.txt", "*.rda", "*.pdf", "*.png", "*._allChromosomeResultsMerged.txt", 
    			"*.txt.gz", "*.vcf.gz", "*.vcf.gz.tbi", "*_chunkedImputationQueue.txt"}
    for _, suffix := range findThese {
    	if saveChunks == false && (suffix == "*.vcf.gz" || suffix == "*.vcf.gz.tbi" || suffix == "*_chunkedImputationQueue.txt") {
    		continue
    	}else{
    		tmpMatches,_ := filepath.Glob(filepath.Join(bindPointTemp, suffix))
    		if len(tmpMatches) != 0 {
    			matches = append(matches,tmpMatches...)
    		}
    	}
    }
    for _,fileTransfer := range matches {
    	fileName := strings.Split(fileTransfer, "/")
    	err := os.Rename(fileTransfer, filepath.Join(outDir,fileName[len(fileName)-1]))
    	if err != nil {
    		fmt.Printf("[func(saveResults) -- transferring final results] Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    	}
    }
    fmt.Printf("[func(saveResults) -- finished transferring final results] Time Elapsed: %.2f minutes\n", time.Since(save).Minutes())
}


func saveQueue (queueFile string, f *os.File) {
	queueFileSave.Lock()
	savedQueue, err := f.WriteString(queueFile+"\n")
	if err != nil {
		fmt.Printf("[func(saveQueue)] There was an error when writing queue to file:\n \t%v\n", err)
		queueFileSave.Unlock()

	}else{
		fmt.Printf("[func(saveQueue)] Saved chunked file to queue list: %v\n", savedQueue)
		f.Sync()
		queueFileSave.Unlock()
	}
}


func usePrevChunks (imputeDir,imputationFileList string) {
	defer wgAllChunks.Done()
	fileQueue, err := os.Open(imputationFileList)
	if err != nil {
		fmt.Printf("[func(usePrevChunks)] There was an error opening the imputation chunk file list. The error is as follows: %v\n", err)
		os.Exit(42)
	}

	scanner := bufio.NewScanner(fileQueue)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		changeQueueSize.Lock()
		processQueue = append(processQueue, scanner.Text())
		changeQueueSize.Unlock()
		fmt.Printf("[func(usePrevChunks)] %s, has been added to the processing queue.\n", scanner.Text())
	}

	allChunksFinished++
}