package main

import (
	"fmt"
	"os/exec"
	"sync"
	"strings"
	"strconv"
	"path/filepath"
	"os"
)

var wgSchedule sync.WaitGroup
var wgAssociation sync.WaitGroup
var wgSummarize sync.WaitGroup
var wgChunk sync.WaitGroup

var queue []string

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
	outDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/biobank_paper_pheWAS/multiple_sclerosis/" # requires last backslash
	imputationFileDir := "/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/TOPMedImputation/" # requires last backslash
	sampleFile :="/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/step0_GRM/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned_relatednessCutoff_0.0625_103154_randomMarkersUsed.sparseGRM.mtx.sampleIDs.txt"
	vcfField :="DS" # DS or GT
	chunkVariants := "1000000"
	chromosomeLengthFile="/home/brunettt/Downloads/software/github_repos/CCPM_Biobank_GWAS_Pipeline/hg38_chrom_sizes.txt"
	//GMMMATmodelFile: ="/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/biobank_paper_pheWAS/multiple_sclerosis/multiple_sclerosis_CCPMbb_freeze_v1.3.rda" # ends in .rda from step 1
	//varianceRatioFile :="/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/biobank_paper_pheWAS/multiple_sclerosis/multiple_sclerosis_CCPMbb_freeze_v1.3.varianceRatio.txt" # ends in .txt from step1
	//outNamePrefix :="GO_TEST_multiple_sclerosis_CCPMbb_freeze_v1.3"

	// end parse: 


	wgSchedule.Add(1)
	go nullModel(bindPoint,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outPrefix,rel,loco,covTransform) 
    wgSchedule.Wait()


 	chroms:= strings.Split(chromosomes, "-")
 	start:= strings.TrimSpace(chroms[0])
 	end:= strings.TrimSpace(chroms[1])
 	
	chunk(start,end,build,chunkVariants,chromosomeLengthFile)

    for i:=start; i < start+1; i++ {
    	wgAssociation.Add(1)
    	go associationAnalysis(bindpoint,container,build,vcfFile,vcfFileIndex,vcfField,sampleIDFile,strconv.Itoa(i),IsDropMissingDosages,MAF,MAC,outPrefix,LOCO)
    }
    wgAssociation.Wait()

 /*   go cleanResults()
    wgSummarize.Wait()
*/
    fmt.Println("FINISHED!!!!!")


}

func nullModel (bindPoint,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outPrefix,rel,loco,covTransform string) {
	defer wgSchedule.Done()

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
}


func associationAnalysis (bindpoint,container,build,vcfFile,vcfFileIndex,vcfField,sampleIDFile,chrom,IsDropMissingDosages,MAF,MAC,outPrefix,LOCO string) {
	defer wgAssociation.Done()
	switch {
	case build == "hg38":
		cmd := exec.Command("singularity", "run", "-B", bindPoint, container, "/usr/lib/R/bin/R/Rscript", "/opt/step2_SPAtests.R",
			"--vcfFile="+vcfFile
			"--vcfFileIndex="+vcfFileIndex,
			"--vcfField="+vcfField,
			"--sampleFile="+sampleIDFile,
			"--chrom=chr"+chrom,
			"--IsDropMissingDosages="+IsDropMissingDosages,
			"--minMAF="+MAF,
			"--minMAC="+MAC,
			"--GMMATmodelFile="+filepath.Join(outDir, outPrefix)+".rda"
			"--varianceRatioFile="+filepath.Join(outDir, outPrefix)+".varianceRatio.txt"
			"--numLinesOutput=2",
			"--IsOutputAFinCaseCtrl=TRUE",
			"--IsOutputHetHomCountsinCaseCtrl=TRUE",
			"--IsOutputBETASEinBurdenTest=TRUE",
			"--SAIGEOutputFile="+filepath.Join(outDir, outPrefix)+"_chrm"+chrom+"_SNPassociationAnalysis.txt",
			"--LOCO="+loco)
		cmd.Run()
		_ := cmd.Wait()
	case build == "hg19":
		cmd := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/step2_SPAtests.R",
			"--vcfFile="+sparseGRM,
			"--vcfFileIndex="+sampleIDFile,
			"--vcfField="+phenoFile,
			"--sampleFile="+plink,
			"--chrom="+chrom,
			"--IsDropMissingDosages="+IsDropMissingDosages,
			"--minMAF="+MAF,
			"--minMAC="+MAC,
			"--GMMATmodelFile="+filepath.Join(outDir, outPrefix)+".rda"
			"--varianceRatioFile="+filepath.Join(outDir, outPrefix)+".varianceRatio.txt"
			"--numLinesOutput=2",
			"--IsOutputAFinCaseCtrl=TRUE",
			"--IsOutputHetHomCountsinCaseCtrl=TRUE",
			"--IsOutputBETASEinBurdenTest=TRUE",
			"--SAIGEOutputFile="+filepath.Join(outDir, outPrefix)+"_chrm"+chrom+"_SNPassociationAnalysis.txt",
			"--LOCO="+loco)
		cmd.Run()
		_ := cmd.Wait()
	default:
		fmt.Println("The build you have selected must be either hg38 or hg19.  Please note, build is defined on the build of
			the imputed data set, not the plink file and not the build you desire your output to be based upon.")
		os.Exit(42)

	}
}


/*func cleanResults () {
	defer wgSummarize.Done()

	cmd := exec.Command("singularity", "run", "-B", bindPoint, container, "/opt/step3_GWASsummary.R"
		"--assocFile=$inputAssociationFile",
		"--infoFile=$infoFile",
		"--dataOutputPrefix=$dataOutputPrefix",
		"--macFilter=10",
		"--mafFilter=0.05",
		"--traitType=binary")
	)
}
*/


func chunk(start,end,build,chunkVariants,chromosomeLengthFile string) {
	fmt.Println("%v\n", chunkVariants)
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
	fmt.Printf("%v\n", startInt)
	fmt.Printf("%v\n",endInt)
	for i:=startInt; i<endInt+1; i++ {
		chromInt:= strconv.Itoa(i)
		fmt.Printf("%v\n", i)
		wgChunk.Add(1)
		go smallerChunk(chromInt,build)
	}
	wgChunk.Wait()


}

func smallerChunk(chrom,build string){
	defer wgChunk.Done()

	if build == "hg38" {
		chrom = "chr"+chrom
	}
	time.Sleep(5 * time.Second)


	fmt.Printf("%s\n", chrom)
	
}



	/* grab vcf header and then do a split call by line
	/homelink/brunettt/TOOLS/bcftools-1.5/bcftools view -h chr1_rsq70_merged_renamed.vcf.gz > header.txt

	grep -v of # and split -l <numberOfLines> --suffix-length=100000 imputationFileName prefix

	cat header to each split

	bgzip -c file.vcf > file.vcf.gz
	tabix -p vcf file.vcf.gz

	rm .vcf non-gzipped version
	submit to queue key is chromosome and value is the for loop replaceding the vcfFile and vcfFileIndex

	rm all subset vcf.gz files -- careful not to remove the imputed files!!

	all this can be a goroutine per chromsome with its own so they run concurrently and a sync object
	*/

