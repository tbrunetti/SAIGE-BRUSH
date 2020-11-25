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
	"io/ioutil"
	"archive/tar"
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

func main() {
	// always need to happen regardless of pipeline step being run
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
	
	
	
	allChunksFinished = 1

	// Before starting pipeline perform a basic input check
	checkInput(parserMap.MAC,parserMap.MAF,parserMap.PhenoFile,parserMap.Pheno,parserMap.Covars,parserMap.SampleID)

	 // create tmp folder to be deleted at end of run
 	err:=os.Mkdir(filepath.Join(parserMap.BindPointTemp, "tmp_saige"), 0755)
 	if err != nil {
 		fmt.Printf("[func(main)] There was an error creating the tmp directory. \t%v\n", err)
 		os.Exit(42)
 	}else {
 		fmt.Printf("[func(main) Created tmp directory called tmp_saige in %s\n]", parserMap.BindPointTemp)
 	}
 	defer os.RemoveAll(filepath.Join(parserMap.BindPointTemp, "tmp_saige")) // remove tmp_saige dir after main finishes

	// create queue file for saving and re-using chunks
	f,_:= os.Create(filepath.Join(parserMap.BindPointTemp, "tmp_saige", parserMap.OutPrefix+"_chunkedImputationQueue.txt"))
	defer f.Close()


	////////////////////////////////////////////////
	// START BUILDING LOGIC FOR PIPELINE START STEPS
	////////////////////////////////////////////////

	// split chroms -- only pretains to option when chunking of imputation file is required
 	chroms:= strings.Split(parserMap.Chromosomes, "-")
 	start:= strings.TrimSpace(chroms[0])
 	end:= strings.TrimSpace(chroms[1])


	// STEP0: Generate Kinship Matrix
	if parserMap.GenerateGRM == true {
		plinkFloat,_ := strconv.ParseFloat(parserMap.GrmMAF, 64)
		plinkFloat = plinkFloat + 0.005
		plinkFloatString := fmt.Sprintf("%f", plinkFloat)
		plinkLD := exec.Command("singularity", "run", "-B", parserMap.BindPoint+","+parserMap.BindPointTemp, parserMap.Container, "/opt/plink2",
			"--bfile",
			parserMap.Plink,
			"--maf",
			plinkFloatString,
			"--make-bed",
			"--out",
			filepath.Join(parserMap.BindPointTemp, "tmp_saige", parserMap.OutPrefix + "_"+parserMap.GrmMAF))
		plinkLD.Stdout = os.Stdout
		plinkLD.Stderr = os.Stderr
		plinkLD.Run()
		
		totalSNPsTmp,_ := exec.Command("singularity", "run", "-B", parserMap.BindPoint+","+parserMap.BindPointTemp, parserMap.Container,
			"wc",
			"-l",
			filepath.Join(parserMap.BindPointTemp, "tmp_saige", parserMap.OutPrefix+"_"+parserMap.GrmMAF+".bim")).Output()

		totalSNPs:=strings.Fields(string(totalSNPsTmp)) // convert byte stream to string and then split based on whitespace to get int string


		fmt.Printf("[func(main) generate GRM] There are a total of %v snps that meet the maf requirements for GRM calculation.\n", totalSNPs[0])


		createGRM := exec.Command("singularity", "run", "-B", parserMap.BindPoint+","+parserMap.BindPointTemp, parserMap.Container, "/usr/lib/R/bin/Rscript", "/opt/createSparseGRM.R",
			"--plinkFile="+parserMap.Plink,
			"--outputPrefix="+filepath.Join(parserMap.BindPointTemp, "tmp_saige", parserMap.OutPrefix),
			"--numRandomMarkerforSparseKin="+string(totalSNPs[0]),
			"--relatednessCutoff="+parserMap.Rel,
			"--memoryChunk=2",
			"--isDiagofKinSetAsOne=FALSE",
			"--nThreads="+parserMap.NThreads,
			"--minMAFforGRM="+parserMap.GrmMAF)
		createGRM.Stdout = os.Stdout
		createGRM.Stderr = os.Stderr
		createGRM.Run()

		parserMap.SparseGRM = filepath.Join(parserMap.BindPointTemp, "tmp_saige", parserMap.OutPrefix+"_relatednessCutoff_"+parserMap.Rel+"_"+string(totalSNPs[0])+"_randomMarkersUsed.sparseGRM.mtx")
		fmt.Printf("[func(main) -- generate GRM] Sparse GRM path located at: %s\n", parserMap.SparseGRM)
		parserMap.SampleIDFile = filepath.Join(parserMap.BindPointTemp, "tmp_saige", parserMap.OutPrefix+"_relatednessCutoff_"+parserMap.Rel+"_"+string(totalSNPs[0])+"_randomMarkersUsed.sparseGRM.mtx.sampleIDs.txt")
		fmt.Printf("[func(main) -- generate GRM] Sparse GRM sampleID path located at: %s\n", parserMap.SampleIDFile)
	}

 	
 	// STEP1: Run Null Model Queue
 	wgNull.Add(1)
 	if parserMap.SkipChunking == true {
		go nullModel(parserMap.BindPoint,parserMap.BindPointTemp,parserMap.Container,parserMap.SparseGRM,parserMap.SampleIDFile,parserMap.PhenoFile,parserMap.Plink,
			parserMap.Trait,parserMap.Pheno,parserMap.InvNorm,parserMap.Covars,parserMap.SampleID,parserMap.NThreads,parserMap.SparseKin,parserMap.Markers,
			parserMap.OutDir,parserMap.OutPrefix,parserMap.Rel,parserMap.Loco,parserMap.CovTransform)
		time.Sleep(1* time.Minute) 
	} else {
		threadsNull,err := strconv.Atoi(parserMap.NThreads)
		if err != nil {
			fmt.Printf("[func(main) null thread allocation] There was an error converting threads: %v\n", err)
			os.Exit(42)
		}
		toNull := math.Ceil(float64(threadsNull) * 0.75)
		toChunk := math.Ceil(float64(threadsNull) - toNull)
		toNullString := fmt.Sprintf("%f", toNull)
		
		fmt.Printf("There are %v threads requested.  %v are reserverd for the null model generation. %v are reserved for chunking.\n", threadsNull, toNull, toChunk)
		
		go nullModel(parserMap.BindPoint,parserMap.BindPointTemp,parserMap.Container,parserMap.SparseGRM,parserMap.SampleIDFile,parserMap.PhenoFile,parserMap.Plink,
			parserMap.Trait,parserMap.Pheno,parserMap.InvNorm,parserMap.Covars,parserMap.SampleID,toNullString,parserMap.SparseKin,parserMap.Markers,parserMap.OutDir,
			parserMap.OutPrefix,parserMap.Rel,parserMap.Loco,parserMap.CovTransform) 
		time.Sleep(1* time.Minute) 

	}
 	
 	/* STEP1a: Chunks Queue or Use of previously chunked imputation data
 	chunk can run the same time as the null model; and the association analysis needs to wait for the 
 	null to finish and chunk to finish*/
 	if parserMap.SkipChunking ==  false {
 		wgAllChunks.Add(1)
    	go chunk(start,end,parserMap.Build,parserMap.OutDir,parserMap.ChromosomeLengthFile,parserMap.ImputeDir,parserMap.ImputeSuffix,parserMap.BindPoint,
    		parserMap.BindPointTemp,parserMap.Container,parserMap.ChunkVariants,f)
 	}else {
 		wgAllChunks.Add(1)
 		go usePrevChunks(parserMap.ImputeDir, parserMap.ImputationFileList)
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
			subName := strings.TrimSuffix(vcfFile, parserMap.ImputeSuffix)
			// if condition is true, that means the data is not chunked and need to be read from the imputeDir not outDir
			if ((tmp[0]+parserMap.ImputeSuffix == vcfFile) || (parserMap.SkipChunking == true)) {
				wgAssociation.Add(1)
				go associationAnalysis(parserMap.BindPoint,parserMap.BindPointTemp,parserMap.Container,filepath.Join(parserMap.ImputeDir,vcfFile),parserMap.VcfField,parserMap.OutDir,
				tmp[0],subName,parserMap.SampleIDFile,parserMap.IsDropMissingDosages,parserMap.OutPrefix,parserMap.Loco)
				processQueue = processQueue[1:]
				time.Sleep(1* time.Second) // prevent queue overload
			}else{
				wgAssociation.Add(1)
				go associationAnalysis(parserMap.BindPoint,parserMap.BindPointTemp,parserMap.Container,filepath.Join(parserMap.BindPointTemp,"tmp_saige",vcfFile),parserMap.VcfField,
				parserMap.OutDir,tmp[0],subName,parserMap.SampleIDFile,parserMap.IsDropMissingDosages,parserMap.OutPrefix,parserMap.Loco)
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
    concat := exec.Command("singularity", "run", "-B", parserMap.BindPoint+","+parserMap.BindPointTemp, parserMap.Container, "/opt/concatenate.sh", filepath.Join(parserMap.BindPointTemp,"tmp_saige",parserMap.OutPrefix))
    concat.Stdout = os.Stdout
    concat.Stderr = os.Stderr
    concat.Run()
    
    fmt.Printf("[func(main) -- concatenate] Finished all association results. Time Elapsed: %.2f minutes\n", time.Since(concatTime).Minutes())

    // STEP3: Clean, Visualize, and Summarize results
    graph := time.Now()
	formatted = fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", graph.Year(), graph.Month(), graph.Day(), graph.Hour(), graph.Minute(), graph.Second())
    fmt.Printf("[func(main) -- clean and graph results] Start data clean up, visualization, and summarization...\n")
    cleanAndGraph := exec.Command("singularity", "run", "-B", parserMap.BindPoint+","+parserMap.BindPointTemp, parserMap.Container, "/usr/lib/R/bin/Rscript", "/opt/step3_GWASsummary.R",
			"--assocFile="+filepath.Join(parserMap.BindPointTemp,"tmp_saige",parserMap.OutPrefix) + "_allChromosomeResultsMerged.txt",
			"--infoFile="+parserMap.InfoFile,
			"--dataOutputPrefix="+filepath.Join(parserMap.BindPointTemp,"tmp_saige",parserMap.OutPrefix),
			"--pheno="+parserMap.Pheno,
			"--covars="+parserMap.Covars,
			"--macFilter="+parserMap.MAC,
			"--mafFilter="+parserMap.MAF,
			"--traitType="+parserMap.Trait,
			"--nThreads="+parserMap.NThreads)
	cleanAndGraph.Stdout = os.Stdout
    cleanAndGraph.Stderr = os.Stderr
    cleanAndGraph.Run()


    fmt.Printf("[func(main) -- clean and graph results] Finished all data clean up, visualizations, and summarization. Time Elapsed: %.2f minutes\n", time.Since(graph).Minutes())

    //TODO: clean up temp files
    saveResults(parserMap.BindPointTemp,parserMap.OutPrefix,parserMap.OutDir,parserMap.SaveChunks)

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
		filepath.Join(bindPointTemp,"tmp_saige",chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix,
		filepath.Join(imputeDir, chrom+imputeSuffix))
	subset.Run()

	errorHandling.Lock()
    subset.Stdout = os.Stdout
    subset.Stderr = os.Stderr
    errorHandling.Unlock()


	index := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/tabix",
		"-p",
		"vcf",
		filepath.Join(bindPointTemp,"tmp_saige",chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)
	index.Run()

	errorHandling.Lock()
   	index.Stdout = os.Stdout
    index.Stderr = os.Stderr
    errorHandling.Unlock()

	totalVariants,_ := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/bcftools",
		"index",
		"--nrecords",
		filepath.Join(bindPointTemp,"tmp_saige",chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix).Output()

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
			"--outputPrefix="+filepath.Join(bindPointTemp,"tmp_saige",outPrefix),
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
			"--outputPrefix="+filepath.Join(bindPointTemp,"tmp_saige",outPrefix),
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
		"--GMMATmodelFile="+filepath.Join(bindPointTemp, "tmp_saige", outPrefix)+".rda",
		"--varianceRatioFile="+filepath.Join(bindPointTemp, "tmp_saige", outPrefix)+".varianceRatio.txt",
		"--numLinesOutput=2",
		"--IsOutputAFinCaseCtrl=TRUE",
		"--IsOutputHetHomCountsinCaseCtrl=TRUE",
		"--IsOutputBETASEinBurdenTest=TRUE",
		"--IsOutputNinCaseCtrl=TRUE",
		"--SAIGEOutputFile="+filepath.Join(bindPointTemp, "tmp_saige", outPrefix)+"_"+subName+"_SNPassociationAnalysis.txt",
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


func checkInput(MAC,MAF,phenoFile,pheno,covars,sampleID string) {
	var lineNumber int
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


	// check phenoFile for covars, sampleID, and trait
	allCovsSplit := strings.Split(covars, ",")
	allCovsSplit = append(allCovsSplit, pheno)
	allCovsSplit = append(allCovsSplit, sampleID)
	checkPhenoFile,err := os.Open(phenoFile)
	if err != nil {
		fmt.Printf("There was an error opening your phenotype file: \n\t%v", err)
	}
	defer checkPhenoFile.Close()

	readFile := bufio.NewScanner(checkPhenoFile)
	for readFile.Scan(){
		if lineNumber == 0 {
			lineNumber++
			headerLine := readFile.Text()
			headerLineSplit := strings.Split(headerLine,  "\t")
			for _,value := range allCovsSplit {
				findElement(headerLineSplit, value)
			}
		} else {
			break
		}
	}
}


func saveResults(bindPointTemp,outputPrefix,outDir string, saveChunks bool) {
	save := time.Now()
	fmt.Printf("[func(saveResults) -- begin transferring final results]\n")
	err:=os.Mkdir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults"), 0755)
	if err != nil {
 		fmt.Printf("[func(saveResults)] There was an error creating the final results directory. \t%v\n", err)
 		os.Exit(42)
 	}else {
 		fmt.Printf("[func(saveResults) Created final results directory called %s\n]", outputPrefix+"_finalResults")
 	}

	matches := make([]string, 0)
    findThese := [14]string{"*.mtx.sampleIDs.txt", "*.sparseGRM.mtx", "*.sparseSigma.mtx", 
    			"*.varianceRatio.txt", "*.rda", "*.pdf", "*.png", "*._allChromosomeResultsMerged.txt", 
    			"*.txt.gz", "*.vcf.gz", "*.vcf.gz.tbi", "*_chunkedImputationQueue.txt","*.log", "*.err"}
    for _, suffix := range findThese {
    	if saveChunks == false && (suffix == "*.vcf.gz" || suffix == "*.vcf.gz.tbi" || suffix == "*_chunkedImputationQueue.txt") {
    		continue
    	}else{
    		tmpMatches,_ := filepath.Glob(filepath.Join(bindPointTemp, "tmp_saige", suffix))
    		if len(tmpMatches) != 0 {
    			matches = append(matches,tmpMatches...)
    		}
    	}
    }
    for _,fileTransfer := range matches {
    	fileName := strings.Split(fileTransfer, "/")
    	err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults",fileName[len(fileName)-1]))
    	if err != nil {
    		fmt.Printf("[func(saveResults) -- transferring final results] Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    	}
    }

    //tar final file
   	filesToTar, err := ioutil.ReadDir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults"))
    tarFileName, tarErr := os.Create(filepath.Join(bindPointTemp,outputPrefix+"_finalResults.tar"))
    if tarErr != nil {
    	fmt.Printf("[func(saveResults) Error encountered when creating compressed tar file \t%v\n]", tarErr)
    }
    defer tarFileName.Close()
    var writeTar io.WriteCloser = tarFileName
    tarFileWriter := tar.NewWriter(writeTar)
    defer tarFileWriter.Close() // close when finished

    for _,fileMeta := range filesToTar {
    	file, err := os.Open(filepath.Join(bindPointTemp,outputPrefix+"_finalResults") + string(filepath.Separator) + fileMeta.Name())
    	defer file.Close()
    	// prepare the tar header
     	header := new(tar.Header)
     	header.Name = file.Name()
     	header.Size = fileMeta.Size()
     	header.Mode = int64(fileMeta.Mode())
     	header.ModTime = fileMeta.ModTime()

     	err = tarFileWriter.WriteHeader(header)
      	if err != nil {
      		fmt.Printf("[func(saveResults) -- transferring final results] Problem writing file to tar.  The following error was encountered: %v\n",err)
      	}
 
      	_, err = io.Copy(tarFileWriter, file)
      	if err != nil {
      		fmt.Printf("[func(saveResults) -- transferring final results] Problem copying file to tar.  The following error was encountered: %v\n",err)
      	} 
    }

    defer os.RemoveAll(filepath.Join(bindPointTemp,outputPrefix+"_finalResults"))

	
    // check if tmp bindpoint is different from outdir; if true then move to outdir, if false keep in tmpDir
    tmpLoc := strings.TrimSpace(strings.TrimSuffix(bindPointTemp, "/"))
	finalLoc := strings.TrimSpace(strings.TrimSuffix(outDir, "/"))

    if tmpLoc  != finalLoc {
    	os.Rename(filepath.Join(bindPointTemp,outputPrefix+"_finalResults.tar"), filepath.Join(outDir,outputPrefix+"_finalResults.tar"))
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
		fmt.Printf("[func(saveQueue)] Saved chunked file to queue list: %v\n", string(savedQueue))
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

	allChunksFinished--
}

func findElement (headerSlice []string, element string) {
	for _,headerVal := range headerSlice {
		if headerVal == element {
			fmt.Printf("[func(findElement)] Confirmed %v in phenoFile\n", element)
			return
		}
	}
	fmt.Printf("[func(findElement)] FAIL -- %v is not listed in phenofile.  Please update phenofile.\n", element)
	os.Exit(42)
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