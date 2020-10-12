package main

import (
	"fmt"
	"os/exec"
	"sync"
	"strings"
	"strconv"
)

var wgSchedule sync.WaitGroup
var wgAssociation sync.WaitGroup

func main() {
	// TODO: parse all this info
	bindPoint := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/"
	container := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/custom_SAIGEbuild_v0.39_CCPM_biobank_07292020.simg"
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
	outPrefix := "GO_TEST_multiple_sclerosis_CCPMbb_freeze_v1.3"
	rel := "0.0625"
	loco := "TRUE"
	covTransform := "TRUE"
	chromosomes := "1-22"
	build := "hg38"
	// end parse


	wgSchedule.Add(1)
	go nullModel(bindPoint,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outPrefix,rel,loco,covTransform) 
    wgSchedule.Wait()

    chroms := string.Split(chromosomes, "-")
    start, err := strconv.Atoi(chroms[0])
    if err != nil {
    	fmt.Printf("Ooops the chromosome start you specified is invalid.\n Must be integer!")
    }
    end, err := strconv.Atoi(chroms[1])
    if err != nil {
    	fmt.Printf("Ooops the chromosome end you specified is invalid.\n Must be integer!")
    }


    for i:=start; i < start+1; i++ {
    	wgAssociation.Add(1)
    	go associationAnalysis()
    }
    wgAssociation.Wait()

    fmt.Println("FINISHED!!!!!")


}

func nullModel (bindPoint,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outPrefix,rel,loco,covTransform string) {
	defer wgSchedule.Done()

	cmd := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/step1_fitNULLGLMM.R",
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
		"--outputPrefix="+outPrefix,
		"--relatednessCutoff="+rel,
		"--LOCO="+loco,
		"--isCovariateTransform="+covTransform)

	cmd.Run()
	//err := cmd.Start()
	//if err != nil {
	//	fmt.Println("Oooops, error!\n %v", err)
	//}
	_ = cmd.Wait()
}


func associationAnalysis () {
	defer wgAssociation.Done()
}