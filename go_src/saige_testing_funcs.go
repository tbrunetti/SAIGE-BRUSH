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
var wgPipeline sync.WaitGroup
var wgSmallChunk sync.WaitGroup
var wgNull sync.WaitGroup

func main() {
	runtime.GOMAXPROCS(2)

	// variables that need to be acquired by parsing through user input
	chunkVariants := 15000000
	chromosomeLengthFile := "/home/brunettt/Downloads/software/github_repos/CCPM_Biobank_GWAS_Pipeline/hg38_chrom_sizes.txt"
	build := "hg38"
	chromosomes := "21-22"
	imputeSuffix := "_rsq70_merged_renamed.vcf.gz"
	imputeDir := "/home/brunettt/Downloads/software/github_repos/CCPM_Biobank_GWAS_Pipeline/imputation_data"
	bindPoint := "/home/brunettt/Downloads/software/github_repos/CCPM_Biobank_GWAS_Pipeline/"
	container := "/home/brunettt/Downloads/software/github_repos/CCPM_Biobank_GWAS_Pipeline/SAIGE_v0.39_CCPM_biobank_singularity_recipe_file_10272020.simg"
	/*
	outDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/test_new_pipeline/"
	sparseGRM := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/step0_GRM/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned_relatednessCutoff_0.0625_103154_randomMarkersUsed.sparseGRM.mtx"
	sampleIDFile := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/step0_GRM/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned_relatednessCutoff_0.0625_103154_randomMarkersUsed.sparseGRM.mtx.sampleIDs.txt"
	phenoFile := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/biobank_paper_pheWAS/pheWAS_CCPMbb_freeze_v1.3.txt" 
	plink := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/LDprunedMEGA/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned"
	outPrefix := "GO_TEST_multiple_sclerosis_CCPMbb_freeze_v1.3"
	plink := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/LDprunedMEGA/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned"
	trait := "binary"
	pheno := "multiple_sclerosis"
	invNorm := "FALSE"
	covars := "PC1,PC2,PC3,PC4,PC5,SAIGE_GENDER,age"
	sampleID := "FULL_BBID"
	nThreads := "24"
	sparseKin := "TRUE"
	markers := "30"
	outPrefix := "GO_TEST_multiple_sclerosis_CCPMbb_freeze_v1.3"
	rel := "0.0625"
	loco := "TRUE"
	covTransform := "TRUE"*/
	// end of variables


 	chroms:= strings.Split(chromosomes, "-")
 	start:= strings.TrimSpace(chroms[0])
 	end:= strings.TrimSpace(chroms[1])

 	/*wgNull.Add(1)
	go nullModel(bindPoint,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outPrefix,rel,loco,covTransform) 
	*/
 	// chunk can run the same time as the null model; and the association analysis needs to wait for the null to finish and chunk to finish
    chunk(start,end,build,chromosomeLengthFile,imputeDir,imputeSuffix,bindPoint,container,chunkVariants)
    fmt.Printf("The final queue is:\n %v\n", processQueue)

}


func chunk(start,end,build,chromosomeLengthFile,imputeDir,imputeSuffix,bindPoint,container string, chunkVariants int) {
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
		go smallerChunk(chromInt,build,imputeDir,imputeSuffix,bindPoint,container,chunkVariants)
	}
	wgChunk.Wait()


}

func smallerChunk(chrom,build,imputeDir,imputeSuffix,bindPoint,container string, chunkVariants int){
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


	// if error is seen, print error and exit out of function, otherwise print the total variants in the vcf file and continue
	if err != nil {
		fmt.Printf("%s overall: Error in total variants call:\n%v", chrom, err)
		return
	} else {
		fmt.Printf("A total of %s variants are in the vcf file for %s\n", strings.TrimSpace(string(totalVariants)),chrom)
	}

	// convert byte slice to string, trim any trailing whitespace of either end and then convert to integer
	varVal,err := strconv.Atoi(strings.TrimSpace(string(totalVariants)))
	if err != nil{
		fmt.Printf("There was an error converting the total variants to an integer. The error encountered is:\n%v\n",err)
		return
	}
	

	// if chunk is larger than total variants in file, do not chunk and just add full imputation file to queue
	if chunkVariants < varVal {
		processQueue = append(processQueue, chrom+imputeSuffix)
		fmt.Printf("%v is queue and %d is variant value\n", processQueue, varVal)
	}else {
		for loopId := 1; loopId < maxLoops + 1; loopId++ {
			wgSmallChunk.Add(1)
			go processing(loopId,chunkVariants,bindPoint,container,chrom,imputeDir,imputeSuffix)
		}
	}
	wgSmallChunk.Wait()

}

func processing (loopId,chunkVariants int, bindPoint,container,chrom,imputeDir,imputeSuffix string) {
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
		filepath.Join(imputeDir, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix,
		filepath.Join(imputeDir, chrom+imputeSuffix))
	subset.Run()

	index := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/tabix",
	"-p",
	"vcf",
	filepath.Join(imputeDir, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)

	index.Run()

	totalVariants,_ := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/bcftools",
		"index",
		"--nrecords",
		filepath.Join(imputeDir, chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix).Output()

				
	varVal,err1 := strconv.Atoi(strings.TrimSpace(string(totalVariants)))
	if err1 != nil{
		fmt.Printf("%s, chunk %s:\n\tThere was an error converting the total variants to an integer. The error encountered is:\n%v\n",chrom,loopNum,err1)
		return
	}
			
	if varVal > 0 {
		processQueue = append(processQueue, chrom+"_"+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)
		t1 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
		fmt.Printf("[func(processing) %s] %s, chunk %s has successfully completed and has been added to the processing queue. Time Elapsed: %.2f minutes\n", formatted,chrom,loopNum, time.Since(t0).Minutes())
	}else{
		fmt.Printf("%s is empty with value %d and will not be added to queue.\n", chrom+"_"+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix, varVal)
	}
}
			

/*func nullModel (bindPoint,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outPrefix,rel,loco,covTransform string) {
	defer wgNull.Done()

	cmd := exec.Command("singularity", "run", "-B", bindPoint, container, "/usr/lib/R/bin/R/Rscript", "/opt/step1_fitNULLGLMM.R",
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
	//err := cmd.Start()
	//if err != nil {
	//	fmt.Println("Oooops, error!\n %v", err)
	//}
	_ = cmd.Wait()
}*/