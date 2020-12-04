package main

import (
	"fmt"
	"runtime"
	"strconv"
	"io"
	"strings"
	"os"
	"bufio"
)


var parserMap = struct {
	Runtype string
	ChunkVariants int // defaults to 1000000
	ChromosomeLengthFile string
	Build string // defaults to hg38
	Chromosomes string //defaults to 1-22
	ImputeSuffix string
	ImputeDir string
	BindPoint string
	BindPointTemp string
	Container string //defaults to SAIGE_v0.39_CCPM_biobank_singularity_recipe_file_11162020.simg
	OutDir string
	OutPrefix string // defaults to myOutput
	SparseGRM string
	SampleIDFile string
	PhenoFile string
	Plink string
	Trait string
	Pheno string
	InvNorm string // defaults to FALSE
	Covars string
	SampleID string
	NThreads string // defaults to use all resources detected
	SparseKin string
	Markers string // defaults to 30
	Rel string // defaults to 0.0625
	Loco string // defaults to TRUE
	CovTransform string
	VcfField string // defaults to DS
	MAF string // defaults to 0.05
	MAC string // defaults to 10
	IsDropMissingDosages string // defaults to FALSE
	InfoFile string 
	SaveChunks bool // defaults to true
	ImputationFileList string
	SkipChunking bool // defaults to false
	GenerateGRM bool // defaults to true
	GrmMAF string // defaults to 0.01
}{
	"FULL",1000000, "", "hg38", "1-22", "","","","","SAIGE_v0.39_CCPM_biobank_singularity_recipe_file_11162020.simg",
	"","myOutput","","","","","","",
	"FALSE","","","","TRUE","30","0.0625","TRUE","TRUE","DS","0.05","10","FALSE", "",
	true,"",false,true,"0.01"}

func main () {
	configFilename := os.Args[1]
	parser(configFilename)

	totalCPUsAvail := runtime.NumCPU()
	fmt.Printf("%v total CPUs available.\n", totalCPUsAvail)
	if parserMap.NThreads == "" {
		parserMap.NThreads = strconv.Itoa(totalCPUsAvail)
		runtime.GOMAXPROCS(totalCPUsAvail)
	} else {
		maxThreads,err := strconv.Atoi(parserMap.NThreads)
		if err != nil{
			fmt.Printf("[func(main) Thread Allocation] There was a problem allocating your threads.  You entered: %v\n, %v", parserMap.NThreads, err)
			os.Exit(42)
		}
		runtime.GOMAXPROCS(maxThreads)
	}

	fmt.Printf("%v total CPUs will be used.\n", parserMap.NThreads)
}

func parser (configFile string) {
	fileBytes, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("There was a problems reading config file.\n")
		os.Exit(42)
	}

	defer fileBytes.Close() // once funciton finished close file
	
	//fmt.Printf("%v", parserMap.ChunkVariants)
	
	scanBytes := bufio.NewReader(fileBytes)
	for {
		line,err := scanBytes.ReadString('\n')
		tmpParse := strings.Split(line, ":")
		switch{
		case strings.TrimSpace(tmpParse[0]) == "ChunkVariants":
			chunkSize,err := strconv.Atoi(strings.TrimSpace(tmpParse[1]))
			if err != nil {
				fmt.Printf("[func(parser)] There was an error converting ChunkVariants to integer: %v\n", err)
				os.Exit(42)
			}
			parserMap.ChunkVariants = chunkSize
		case strings.TrimSpace(tmpParse[0]) == "SaveChunks":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.SaveChunks = false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.SaveChunks = true
			} else {
				fmt.Printf("[func(parser)] %v is not a valid option for SaveChunks.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(42)
			}
		case strings.TrimSpace(tmpParse[0]) == "SkipChunking":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.SkipChunking = false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.SkipChunking = true
			} else {
				fmt.Printf("[func(parser)] %v is not a valid option for SkipChunking.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(42)
			}
		case strings.TrimSpace(tmpParse[0]) == "GenerateGRM":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.GenerateGRM = false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.GenerateGRM = true
			} else {
				fmt.Printf("[func(parser)] %v is not a valid option for GenerateGRM.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(42)
			}
		case strings.TrimSpace(tmpParse[0]) == "Runtype":
			parserMap.Runtype = strings.ToUpper(strings.TrimSpace(tmpParse[1]))
		case strings.TrimSpace(tmpParse[0]) == "ChromosomeLengthFile":
			parserMap.ChromosomeLengthFile = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "Build":
			parserMap.Build = strings.ToLower(strings.TrimSpace(tmpParse[1]))
		case strings.TrimSpace(tmpParse[0]) == "Chromosomes":
			parserMap.Chromosomes = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "ImputeSuffix":
			parserMap.ImputeSuffix = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "ImputeDir":
			parserMap.ImputeDir = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "BindPoint":
			parserMap.BindPoint = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "BindPointTemp":
			parserMap.BindPointTemp = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "Container":
			parserMap.Container = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "OutDir":
			parserMap.OutDir = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "OutPrefix":
			parserMap.OutPrefix= strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "SparseGRM":
			parserMap.SparseGRM= strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "SampleIDFile":
			parserMap.SampleIDFile= strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "PhenoFile":
			parserMap.PhenoFile= strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "Plink":
			parserMap.Plink= strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "Trait":
			parserMap.Trait= strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "Pheno":
			parserMap.Pheno = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "InvNorm":
			parserMap.InvNorm = strings.ToUpper(strings.TrimSpace(tmpParse[1]))
		case strings.TrimSpace(tmpParse[0]) == "Covars":
			parserMap.Covars = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "SampleID":
			parserMap.SampleID = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "NThreads":
			parserMap.NThreads = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "SparseKin":
			parserMap.SparseKin = strings.ToUpper(strings.TrimSpace(tmpParse[1]))
		case strings.TrimSpace(tmpParse[0]) == "Markers":
			parserMap.Markers = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "Rel":
			parserMap.Rel = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "Loco":
			parserMap.Loco = strings.ToUpper(strings.TrimSpace(tmpParse[1]))
		case strings.TrimSpace(tmpParse[0]) == "CovTransform":
			parserMap.CovTransform = strings.ToUpper(strings.TrimSpace(tmpParse[1]))
		case strings.TrimSpace(tmpParse[0]) == "VcfField":
			parserMap.VcfField= strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "MAF":
			parserMap.MAF = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "MAC":
			parserMap.MAC = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "IsDropMissingDosages":
			parserMap.IsDropMissingDosages = strings.ToUpper(strings.TrimSpace(tmpParse[1]))
		case strings.TrimSpace(tmpParse[0]) == "InfoFile":
			parserMap.InfoFile = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "ImputationFileList":
			parserMap.ImputationFileList = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "GrmMAF":
			parserMap.GrmMAF = strings.TrimSpace(tmpParse[1])
		}
		if err == io.EOF {
			fmt.Println("[func(parser)] Finished parsing config file!\n")
			break
		}
	}
}

/* bare minimum config file (i.e. these do not have defaults)
chromosomeLengthFile:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/hg38_chrom_sizes.txt
imputeSuffix:_rsq70_merged_renamed.vcf.gz
imputeDir:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/TOPMedImputation
bindPoint:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/
bindPointTemp:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/tmp/
outDir:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/test_new_pipeline/
sparseGRM:
sampleIDFile:
phenoFile:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/biobank_paper_pheWAS/pheWAS_CCPMbb_freeze_v1.3.txt 
plink:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/LDprunedMEGA/Biobank.v1.3.eigenvectors.070620.reordered.LDpruned
trait:binary
pheno:multiple_sclerosis
covars:PC1,PC2,PC3,PC4,PC5,SAIGE_GENDER,age
sampleID:FULL_BBID
infoFile:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/requiredData/TOPMedImputationInfo/allAutosomes.rsq70.info.SAIGE.txt
imputationFileList:/gpfs/scratch/brunettt/test_SAIGE/newSAIGE_test_07262020/test_new_pipeline/GO_TEST_multiple_sclerosis_CCPMbb_freeze_v1.3_chunkedImputationQueue.txt
*/