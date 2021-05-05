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
	SaveAsTar bool // defaults as false
	GenerateNull bool //defaults as true -- need to implement
	GenerateAssociations bool //defaults as true -- need to implement
	GenerateResults bool //defaults as true -- need to implement
	NullModelFile string
	VarianceRatioFile string
	AssociationFile string
}{
	"FULL",1000000, "", "hg38", "1-22", "","","","","SAIGE_v0.39_CCPM_biobank_singularity_recipe_file_11162020.simg",
	"","myOutput","","","","","","",
	"FALSE","","","","TRUE","30","0.0625","TRUE","TRUE","DS","0.05","10","FALSE", "",
	true,"",false,true,"0.01",false, true, true, true, "","",""}

func main() {
	// always need to happen regardless of pipeline step being run
	configFilename := os.Args[1]
	parser(configFilename)

	totalCPUsAvail := runtime.NumCPU()
	fmt.Printf("[func(main) Thread Allocation %s] -- UPDATE! -- %v total CPUs available.\n", time.Now(),totalCPUsAvail)
	if parserMap.NThreads == "" {
		parserMap.NThreads = strconv.Itoa(totalCPUsAvail)
		runtime.GOMAXPROCS(totalCPUsAvail)
	}else {
		maxThreads,err := strconv.Atoi(parserMap.NThreads)
		if err != nil{
			fmt.Printf("[func(main) Thread Allocation %s] -- ERROR! -- There was a problem allocating your threads.  You entered: %v\n, %v", time.Now(),parserMap.NThreads, err)
			os.Exit(42)
		}
		runtime.GOMAXPROCS(maxThreads)
	}

	fmt.Printf("[func(main) Thread Allocation %s] -- UPDATE! -- %v total CPUs will be used.\n", time.Now(), parserMap.NThreads)
	
	
	
	allChunksFinished = 1

	// Before starting pipeline perform a basic input check
	checkInput(parserMap.MAC,parserMap.MAF,parserMap.PhenoFile,parserMap.Pheno,parserMap.Covars,parserMap.SampleID)

	 // create tmp folder to be deleted at end of run
 	err:=os.Mkdir(filepath.Join(parserMap.BindPointTemp, "tmp_saige"), 0755)
 	if err != nil {
 		fmt.Printf("[func(main) %s] -- ERROR! -- There was an error creating the tmp directory. \t%v\n", time.Now(), err)
 		os.Exit(42)
 	}else {
 		fmt.Printf("[func(main) %s] -- UPDATE! -- Created tmp directory called tmp_saige in %s\n", time.Now(), parserMap.BindPointTemp)
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


		fmt.Printf("[func(main) generate GRM %s] -- UPDATE! -- There are a total of %v snps that meet the maf requirements for GRM calculation.\n", time.Now(),totalSNPs[0])


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
		fmt.Printf("[func(main) -- generate GRM %s] -- UPDATE! -- Sparse GRM path located at: %s\n", time.Now(),parserMap.SparseGRM)
		parserMap.SampleIDFile = filepath.Join(parserMap.BindPointTemp, "tmp_saige", parserMap.OutPrefix+"_relatednessCutoff_"+parserMap.Rel+"_"+string(totalSNPs[0])+"_randomMarkersUsed.sparseGRM.mtx.sampleIDs.txt")
		fmt.Printf("[func(main) -- generate GRM %s] -- UPDATE! -- Sparse GRM sampleID path located at: %s\n", time.Now(),parserMap.SampleIDFile)
	} else {
		if _,err := os.Stat(parserMap.SparseGRM); err == nil {
			fmt.Printf("[func(main) -- skip GRM; use supplied check] -- CONFIRMED! -- The path and file are reachable for config variable SparseGRM.\n")
		} else if os.IsNotExist(err) {
			fmt.Printf("[func(main) -- skip GRM; use supplied check] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable SparseGRM.\n", parserMap.SparseGRM)
			os.Exit(42)
		} else {
			fmt.Printf("[func(main) -- skip GRM; use supplied check] -- WARNING! -- File may exist from variable SparseGRM but the following error occurred: %v The path and file are reachable for config variable SparseGRM.\n", err)
		}
		}
		
		if _,err := os.Stat(parserMap.SampleIDFile); err == nil {
			fmt.Printf("[func(main) -- skip GRM; use supplied check] -- CONFIRMED! -- The path and file are reachable for config variable SampleIDFile.\n")
		} else if os.IsNotExist(err) {
			fmt.Printf("[func(main) -- skip GRM; use supplied check] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable SampleIDFile.\n", parserMap.SampleIDFile)
			os.Exit(42)
		} else {
			fmt.Printf("[func(main) -- skip GRM; use supplied check] -- WARNING! -- File may exist from variable SampleIDFile but the following error occurred: %v The path and file are reachable for config variable SampleIDFile.\n", err)
		}	

 	
 	// STEP1: Run Null Model Queue
 	if parserMap.GenerateNull == true {
	 	wgNull.Add(1)
	 	if parserMap.SkipChunking == true {
			go nullModel(parserMap.BindPoint,parserMap.BindPointTemp,parserMap.Container,parserMap.SparseGRM,parserMap.SampleIDFile,parserMap.PhenoFile,parserMap.Plink,
				parserMap.Trait,parserMap.Pheno,parserMap.InvNorm,parserMap.Covars,parserMap.SampleID,parserMap.NThreads,parserMap.SparseKin,parserMap.Markers,
				parserMap.OutDir,parserMap.OutPrefix,parserMap.Rel,parserMap.Loco,parserMap.CovTransform)
			time.Sleep(1* time.Minute) 
		} else {
			threadsNull,err := strconv.Atoi(parserMap.NThreads)
			if err != nil {
				fmt.Printf("[func(main) null thread allocation %s] -- ERROR! -- There was an error converting threads: %v\n", time.Now(),err)
				os.Exit(42)
			}
			toNull := math.Ceil(float64(threadsNull) * 0.75)
			toChunk := math.Ceil(float64(threadsNull) - toNull)
			toNullString := fmt.Sprintf("%f", toNull)
			
			fmt.Printf("func(main) null thread allocation %s] -- UPDATE! -- There are %v threads requested.  %v are reserverd for the null model generation. %v are reserved for chunking.\n", time.Now(), threadsNull, toNull, toChunk)
			
			go nullModel(parserMap.BindPoint,parserMap.BindPointTemp,parserMap.Container,parserMap.SparseGRM,parserMap.SampleIDFile,parserMap.PhenoFile,parserMap.Plink,
				parserMap.Trait,parserMap.Pheno,parserMap.InvNorm,parserMap.Covars,parserMap.SampleID,toNullString,parserMap.SparseKin,parserMap.Markers,parserMap.OutDir,
				parserMap.OutPrefix,parserMap.Rel,parserMap.Loco,parserMap.CovTransform) 
			time.Sleep(1* time.Minute) 

		}
	}else if (parserMap.GenerateNull == false) && (parserMap.GenerateAssociations==true) {
		if _,err := os.Stat(parserMap.NullModelFile); err == nil {
			fmt.Printf("[func(main) -- skip null model file; use supplied check] -- CONFIRMED! -- The path and file are reachable for config variable NullModelFile.\n")
			}else if os.IsNotExist(err) {
				fmt.Printf("[func(main) -- skip null model file; use supplied check] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable NullModelFile.\n", parserMap.VarianceRatioFile)
				os.Exit(42)
			}else {
				fmt.Printf("[func(main) -- skip null model file; use supplied check] -- WARNING! -- File may exist from variable NullModelFile but the following error occurred: %v \n", err)
			}
		if _,err := os.Stat(parserMap.VarianceRatioFile); err == nil {
			fmt.Printf("[func(main) -- skip variance ratio calculation; use supplied check] -- CONFIRMED! -- The path and file are reachable for config variable VarianceRatioFile.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(main) -- skip variance ratio calculation; use supplied check] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable VarianceRatioFile.\n", parserMap.VarianceRatioFile)
				os.Exit(42)
			}else {
				fmt.Printf("[func(main) -- skip variance ratio calculation; use supplied check] -- WARNING! -- File may exist from variable VariableRatioFile but the following error occurred: %v \n", err)
			}
	}else {
		fmt.Printf("[func(main)] -- UPDATE! -- null model not needed since no association analyses will be calculated.\n")
	}
 	
 	
 	/* STEP1a: Chunks Queue or Use of previously chunked imputation data
 	chunk can run the same time as the null model; and the association analysis needs to wait for the 
 	null to finish and chunk to finish*/
 	if ((parserMap.SkipChunking ==  false) && (parserMap.GenerateAssociations == true)) {
 		fmt.Printf("[func(main) Queue status] -- UPDATE! -- Chunking files for queue for association analyses\n")
 		wgAllChunks.Add(1)
    	go chunk(start,end,parserMap.Build,parserMap.OutDir,parserMap.ChromosomeLengthFile,parserMap.ImputeDir,parserMap.ImputeSuffix,parserMap.BindPoint,
    		parserMap.BindPointTemp,parserMap.Container,parserMap.ChunkVariants,f)
 	}else if ((parserMap.SkipChunking ==  true) && (parserMap.GenerateAssociations == true)) {
 		fmt.Printf("[func(main) Queue status] -- UPDATE! -- Reusing previously chunked and index files for queue for association analyses\n")
 		wgAllChunks.Add(1)
 		go usePrevChunks(start, end, parserMap.Build, parserMap.ImputeDir, parserMap.ImputationFileList)
 	}else {
 		fmt.Printf("[func(main) Queue status] -- UPDATE! -- Skipping queue since no associations are required\n")
 	}
  
    
    // wait for null model to finish before proceeding with association analysis -- no need to wait for chunk to finish; 
    // if GenerateNull is set to false, wgNull sync will already be 0 since no jobs addded so will move past this command
    wgNull.Wait()

	
    /* STEP2: Association Analysis Queue
	while loop to keep submitting jobs until queue is empty and no more subsets are available*/
	if parserMap.GenerateAssociations == true {
		for allChunksFinished == 1 || len(processQueue) != 0 {
			if ((allAssocationsRunning < totalCPUsAvail) && (len(processQueue) > 0)) {
				fmt.Printf("[func(main) %s] -- UPDATE! -- The total number of associations running is %v.\n", time.Now(), allAssocationsRunning)
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
	    fmt.Printf("[func(main) %s] -- UPDATE! -- Concatenating all association results...\n", formatted)
	    concat := exec.Command("singularity", "run", "-B", parserMap.BindPoint+","+parserMap.BindPointTemp, parserMap.Container, "/opt/concatenate.sh", filepath.Join(parserMap.BindPointTemp,"tmp_saige",parserMap.OutPrefix))
	    concat.Stdout = os.Stdout
	    concat.Stderr = os.Stderr
	    concat.Run()
	    
	    fmt.Printf("[func(main) -- concatenate %s] -- UPDATE! -- Finished all association results. Time Elapsed: %.2f minutes\n", time.Now(),time.Since(concatTime).Minutes())
	}
    
    // STEP3: Clean, Visualize, and Summarize results
    if parserMap.GenerateResults == true {
    	if parserMap.GenerateAssociations == false {
			if _,err := os.Stat(parserMap.AssociationFile); err == nil {
				fmt.Printf("[func(main) -- skip associationAnalysis; use supplied check] -- CONFIRMED! -- The path and file are reachable for config variable AssociationFile.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(main) -- skip associationAnalysis; use supplied check] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable AssociationFile.\n", parserMap.AssociationFile)
				os.Exit(42)
			} else {
				fmt.Printf("[func(main) -- skip associationAnalysis; use supplied check] -- WARNING! -- File may exist from variable AssociationFile but the following error occurred: %v \n", err)
			}
		} else{
    		parserMap.AssociationFile = filepath.Join(parserMap.BindPointTemp,"tmp_saige",parserMap.OutPrefix) + "_allChromosomeResultsMerged.txt"
    	}	
	    graph := time.Now()
		//formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", graph.Year(), graph.Month(), graph.Day(), graph.Hour(), graph.Minute(), graph.Second())
	    fmt.Printf("[func(main) -- clean and graph results %s] Start data clean up, visualization, and summarization...\n", time.Now())
	    cleanAndGraph := exec.Command("singularity", "run", "-B", parserMap.BindPoint+","+parserMap.BindPointTemp, parserMap.Container, "/usr/lib/R/bin/Rscript", "/opt/step3_GWASsummary.R",
				"--assocFile="+parserMap.AssociationFile,
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

	    fmt.Printf("[func(main) -- clean and graph results %s] -- UPDATE! -- Finished all data clean up, visualizations, and summarization. Time Elapsed: %.2f minutes\n", time.Now(), time.Since(graph).Minutes())
	}

    //TODO: clean up temp files
    saveResults(parserMap.BindPointTemp,parserMap.OutPrefix,parserMap.OutDir,parserMap.SaveChunks,parserMap.SaveAsTar)

    // Pipeline Finish
    fmt.Printf("[func(main) %s] -- UPDATE! -- All threads are finished and pipeline is complete!\n", time.Now())
}


func chunk(start,end,build,outDir,chromosomeLengthFile,imputeDir,imputeSuffix,bindPoint,bindPointTemp,container string, chunkVariants int, f *os.File) {
	defer wgAllChunks.Done() //once function finishes decrement sync object

	fileBytes, err := os.Open(chromosomeLengthFile)
	if err != nil {
		fmt.Printf("[func(chunk) %s] -- ERROR! -- There was a problems reading in the chromosome length file.", time.Now())
		os.Exit(10)
	}


	defer fileBytes.Close() // once funciton finished close file

	scanBytes := bufio.NewReader(fileBytes)
	//var line string
	for {
		line, err := scanBytes.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Println("[func(chunk)] -- ERROR! -- An error occurred when reading in the chromosome length file.")
			os.Exit(10)
		}

		if err == io.EOF {
			fmt.Println("[func(chunk)] -- UPDATE! -- Finished reading chromosome file length")
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
		wgChunk.Wait() // one chromosome chunk at a time to limit files being open
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
		fmt.Printf("[func(smallerChunk) %s] -- WARNING! -- %s overall: Error in total variants call:\n%v", formatted,chrom, err)
	} else {
		tErr := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", tErr.Year(), tErr.Month(), tErr.Day(),tErr.Hour(), tErr.Minute(), tErr.Second())
		fmt.Printf("[func(smallerChunk) %s] -- UPDATE! -- A total of %s variants are in the vcf file for %s\n", formatted,strings.TrimSpace(string(totalVariants)),chrom)
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
		fmt.Printf("[func(smallerChunk) %s] -- UPDATE! -- %s is in the queue and %d is variant value\n", formatted,chrom+imputeSuffix,varVal)
	}else {
		for loopId := 1; loopId < maxLoops + 1; loopId++ {
			wgSmallChunk.Add(1)
			go processing(loopId,chunkVariants,bindPoint,bindPointTemp,container,chrom,outDir,imputeDir,imputeSuffix,f)
			time.Sleep(1* time.Second)
		}
		wgSmallChunk.Wait()
	}
	
}

func processing (loopId,chunkVariants int, bindPoint,bindPointTemp,container,chrom,outDir,imputeDir,imputeSuffix string, f *os.File) {
	defer wgSmallChunk.Done()
	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(processing) %s] -- UPDATE! -- Processing %s, chunk %d\n", formatted, chrom, loopId)
	
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
	bcfToolsErr := subset.Run()
	if bcfToolsErr != nil {
		fmt.Printf("[func(processing) %s] -- WARNING! -- BcfTools subsetting of %s finished with error: %v\n", time.Now(), chrom+":"+lowerValStr+"-"+upperValStr, bcfToolsErr)
		return
	}

	index := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/tabix",
		"-p",
		"vcf",
		filepath.Join(bindPointTemp,"tmp_saige",chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)
	indexErr:=index.Run()
	if indexErr != nil {
		fmt.Printf("[func(processing) %s] -- WARNING! -- Indexing of %s finished with error: %v\n", time.Now(), chrom+":"+lowerValStr+"-"+upperValStr, indexErr)
		return
	}

	totalVariants,_ := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/bcftools",
		"index",
		"--nrecords",
		filepath.Join(bindPointTemp,"tmp_saige",chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix).Output()


	//fmt.Printf("%v, %s chunk %s, %s-%s", strings.TrimSpace(string(totalVariants)), chrom, loopNum, lowerValStr, upperValStr)				
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
		fmt.Printf("[func(processing) %s] -- UPDATE! -- %s, chunk %s has successfully completed and has been added to the processing queue. Time Elapsed: %.2f minutes\n", formatted,chrom,loopNum, time.Since(t0).Minutes())
	}else{
		t1 := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
		fmt.Printf("[func(processing) %s] -- WARNING! -- %s is empty with value %d and will not be added to queue.\n", formatted, chrom+"_"+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix, varVal)
	}
}
	
func nullModel (bindPoint,bindPointTemp,container,sparseGRM,sampleIDFile,phenoFile,plink,trait,pheno,invNorm,covars,sampleID,nThreads,sparseKin,markers,outDir,outPrefix,rel,loco,covTransform string) {
	defer wgNull.Done() // decrement wgNull sync object
	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(nullModel) %s] -- UPDATE! -- Starting Null Model...\n", formatted)

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

		cmd.Stdout = os.Stdout
    	cmd.Stderr = os.Stderr
		cmd.Run() // run automatically wait for null to finish before processing next lines within function
		parserMap.NullModelFile = filepath.Join(bindPointTemp, "tmp_saige", outPrefix)+".rda"
		parserMap.VarianceRatioFile = filepath.Join(bindPointTemp, "tmp_saige", outPrefix)+".varianceRatio.txt"
  

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

		//errorHandling.Lock()
    	cmd.Stdout = os.Stdout
    	cmd.Stderr = os.Stderr
    	//errorHandling.Unlock()

		cmd.Run() // run automatically wait for null to finish before processing next lines within function

	t1 := time.Now()
	formatted = fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
	fmt.Printf("[func(nullModel) %s] -- UPDATE! -- Finished Null Model. Time Elapsed: %.2f minutes\n", formatted, time.Since(t0).Minutes())
	}
}

func associationAnalysis(bindpoint,bindPointTemp,container,vcfFile,vcfField,outDir,chrom,subName,sampleIDFile,IsDropMissingDosages,outPrefix,loco string) {
	defer wgAssociation.Done() // decrement wgAssociation sync object when function finishes
	queueCheck.Lock() // lock the number of associations running to prevent collision
	allAssocationsRunning++
	queueCheck.Unlock() // unlock shared variable safely

	t0 := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t0.Year(), t0.Month(), t0.Day(),t0.Hour(), t0.Minute(), t0.Second())
	fmt.Printf("[func(associationAnalysis) %s] -- UPDATE! -- Association of %s...\n", formatted, vcfFile)


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
		"--GMMATmodelFile="+parserMap.NullModelFile,
		"--varianceRatioFile="+parserMap.VarianceRatioFile,
		"--numLinesOutput=2",
		"--IsOutputAFinCaseCtrl=TRUE",
		"--IsOutputHetHomCountsinCaseCtrl=TRUE",
		"--IsOutputBETASEinBurdenTest=TRUE",
		"--IsOutputNinCaseCtrl=TRUE",
		"--SAIGEOutputFile="+filepath.Join(bindPointTemp, "tmp_saige", outPrefix)+"_"+subName+"_SNPassociationAnalysis.txt",
		"--LOCO="+loco)
	
	cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
	cmd.Run()

	t1 := time.Now()
	formatted = fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t1.Year(), t1.Month(), t1.Day(),t1.Hour(), t1.Minute(), t1.Second())
	fmt.Printf("[func(associationAnalysis) %s] -- UPDATE! -- %s has completed. Time Elapsed: %.2f minutes\n", formatted,vcfFile,time.Since(t0).Minutes())

	queueCheck.Lock() // lock shared variable to prevent collision
	allAssocationsRunning--
	queueCheck.Unlock() // unlock access to shared variable
}

func checkInput(MAC,MAF,phenoFile,pheno,covars,sampleID string) {
	var lineNumber int
	var sampleIDloc int
	checkIDs := make(map[string]bool)

	// check container path exists
	if _,err := os.Stat(parserMap.Container); err == nil {
		fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path and file are reachable for config variable Container.\n")
	} else if os.IsNotExist(err) {
		fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable Container.\n", parserMap.Container)
		os.Exit(5)
	} else {
		fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable Container but the following error occurred: %v \n", err)
	}

	// check bindpoint exists
	if _,err := os.Stat(parserMap.BindPoint); err == nil {
		fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path and file are reachable for config variable BindPoint.\n")
	} else if os.IsNotExist(err) {
		fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable BindPoint.\n", parserMap.BindPoint)
		os.Exit(5)
	} else {
		fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable BindPoint but the following error occurred: %v \n", err)
	}

	// check tmp bindpoint exists
	if _,err := os.Stat(parserMap.BindPointTemp); err == nil {
		fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path and file are reachable for config variable BindPointTemp.\n")
	} else if os.IsNotExist(err) {
		fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable BindPointTemp.\n", parserMap.BindPointTemp)
		os.Exit(5)
	} else {
		fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable BindPointTemp but the following error occurred: %v \n", err)
	}

	
	// check MAC if float and is between 0.0-0.50
	checkMAC,err := strconv.ParseFloat(MAF, 64)
	if err != nil {
		fmt.Printf("[func(checkInput)] -- ERROR! -- There was an error converting MAC to float 64. See following : %v\n", err)
		os.Exit(5)
	} else if checkMAC < 0.0 {
		fmt.Printf("[func(checkInput)] -- ERROR! -- minor allele count cannot be smaller (negative) than 0 (0%). Please select a positive value.\n")
		os.Exit(5)
	}

	// check MAF is float and is between 0.0-0.50
	checkMAF,err := strconv.ParseFloat(MAF, 64)
	if err != nil {
		fmt.Printf("[func(checkInput)] -- ERROR! -- There was an error converting MAF to float 64. See following : %v\n", err)
		os.Exit(5)
	} else if checkMAF > 0.50 {
		fmt.Printf("[func(checkInput)] -- ERROR! -- minor allele frequency cannot be larger than 0.50 (50%). Please select a value between 0.0-0.50.\n")
		os.Exit(5)
	} else if checkMAF < 0.0 {
		fmt.Printf("[func(checkInput)] -- ERROR! -- minor allele frequency cannot be smaller (negative) than 0 (0%). Please select a value between 0.0-0.50.\n")
		os.Exit(5)
	}

	// check GrmMAF is float and is between 0.0-0.50
	checkGrmMAF,err := strconv.ParseFloat(parserMap.GrmMAF, 64)
	if err != nil {
		fmt.Printf("[func(checkInput)] -- ERROR! -- There was an error converting GrmMaf to float 64. See following : %v\n", err)
		os.Exit(5)
	} else if checkGrmMAF > 0.50 {
		fmt.Printf("[func(checkInput)] -- ERROR! -- minor allele frequency cannot be larger than 0.50 (50%) for GrmMaf variable. Please select a value between 0.0-0.50.\n")
		os.Exit(5)
	} else if checkGrmMAF < 0.0 {
		fmt.Printf("[func(checkInput)] -- ERROR! -- minor allele frequency cannot be smaller (negative) than 0 (0%) for GrmMaf variable. Please select a value between 0.0-0.50.\n")
		os.Exit(5)
	}

	// check that rel is float convertable and between 0.0 -1.0
	checkRel,err := strconv.ParseFloat(parserMap.Rel, 64)
	if err != nil {
		fmt.Printf("[func(checkInput)] -- ERROR! -- There was an error converting Rel to float 64. See following : %v\n", err)
		os.Exit(5)
	} else if checkRel > 1.0 {
		fmt.Printf("[func(checkInput)] -- ERROR! -- Kinship relatedness threshold cannot be larger than 1.0 (100%) for Rel variable. Please select a value between 0.0-1.0.\n")
		os.Exit(5)
	} else if checkRel < 0.0 {
		fmt.Printf("[func(checkInput)] -- ERROR! -- Kinship relatedness threshold cannot be smaller (negative) than 0 (0%) for Rel variable. Please select a value between 0.0-1.0.\n")
		os.Exit(5)
	}

	// check trait is either binary or quantitative
	if ((strings.ToLower(parserMap.Trait) == "binary") || (strings.ToLower(parserMap.Trait) == "quantitative")) {
		fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Config variable Trait is set to %s.\n", parserMap.Trait)
	} else {
		fmt.Printf("func(checkInput)] -- ERROR! -- Please select trait type as either binary or quantitative.  You entered: %s.\n", parserMap.Trait)
		os.Exit(5)		
	}

	// check invNorm is a string boolean
	if ((strings.ToLower(parserMap.InvNorm) == "TRUE") || (parserMap.InvNorm == "FALSE")) {
		fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Config variable InvNorm is set to %s.\n", parserMap.InvNorm)
	} else {
		fmt.Printf("func(checkInput)] -- ERROR! -- Please select InvNorm as either True or False.  You entered: %s.\n", parserMap.InvNorm)
		os.Exit(5)		
	}

	// check SparseKin is a string boolean
	if ((parserMap.SparseKin == "TRUE") || (parserMap.SparseKin == "FALSE")) {
		fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Config variable sparseKin is set to %s.\n", parserMap.SparseKin)
	} else {
		fmt.Printf("func(checkInput)] -- ERROR! -- Please select SparseKin as either True or False.  You entered: %s.\n", parserMap.SparseKin)
		os.Exit(5)		
	}

	// check markers is an integer-based string
	_,err =  strconv.Atoi(parserMap.Markers)
	if err != nil {
		fmt.Printf("func(checkInput)] -- ERROR! -- There was an error converting the Markers variable to an integer. Please ensure this is an integer value. You entered: %s.\n", parserMap.Markers)
		os.Exit(5)		
	}

	// check loco is a string boolean
	if ((parserMap.Loco == "TRUE") || (parserMap.Loco == "FALSE")) {
		fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Config variable Loco is set to %s.\n", parserMap.Loco)
	} else {
		fmt.Printf("func(checkInput)] -- ERROR! -- Please set Loco as either True or False.  You entered: %s.\n", parserMap.Loco)
		os.Exit(5)		
	}

	// check CovTransform is a string boolean
	if ((parserMap.CovTransform == "TRUE") || (parserMap.CovTransform == "FALSE")) {
		fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Config variable CovTransform is set to %s.\n", parserMap.CovTransform)
	} else {
		fmt.Printf("func(checkInput)] -- ERROR! -- Please set CovTransform as either True or False.  You entered: %s.\n", parserMap.CovTransform)
		os.Exit(5)		
	}


	// check phenoFile for covars, sampleID, and trait
	allCovsSplit := strings.Split(covars, ",")
	allCovsSplit = append(allCovsSplit, pheno)
	allCovsSplit = append(allCovsSplit, sampleID)
	checkPhenoFile,err := os.Open(phenoFile)
	if err != nil {
		fmt.Printf("[func(checkInput)] -- ERROR! -- There was an error opening your phenotype file: \n\t%v", err)
		os.Exit(5)
	}
	defer checkPhenoFile.Close()

	// check for duplicate sample IDs
	readFile := bufio.NewScanner(checkPhenoFile)
	for readFile.Scan(){
		if lineNumber == 0 {
			lineNumber++
			headerLine := readFile.Text()
			headerLineSplit := strings.Split(headerLine,  "\t")
			for _,value := range allCovsSplit {
				findElement(headerLineSplit, value)
			}
			for idx,value :=range headerLineSplit{
				if value == parserMap.SampleID{
					sampleIDloc = idx
					break
				}
			}
		} else {
			break
		}
	}

	// have pointer move to beginning of file -- faster than closing and opening the file
	checkPhenoFile.Seek(0, io.SeekStart)
	checkUniq := bufio.NewScanner(checkPhenoFile)
	for checkUniq.Scan(){
		line := checkUniq.Text()
		tmpParse := strings.Split(line, "\t")
		if checkIDs[tmpParse[sampleIDloc]] == true {
			fmt.Printf("[func(checkInput)] -- ERROR! -- Duplicate sample ID detected: %v. Duplicate IDs are not allowed.\n", tmpParse[sampleIDloc])
			os.Exit(5)
		} else {
			checkIDs[tmpParse[sampleIDloc]] = true
		}
	}
	fmt.Printf("[func(checkInput)] -- UPDATE! -- There are a total of %d unique sample IDs.\n", len(checkIDs))


	// if using GenerateGRM or GenerateNull must have plink file and check if it exists
	if ((parserMap.GenerateGRM == true) || (parserMap.GenerateNull == true)) {
		// check plink path
		if _,err := os.Stat(parserMap.Plink + ".bed"); err == nil {
				fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path and file are reachable for config variable Plink.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable Plink.\n", parserMap.Plink)
				os.Exit(5)
			} else {
				fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable Plink but the following error occurred: %v \n", err)
			}
	}

	// if using GenerateNull is set to true but reusing an existing GRM, then check these files exist
	if ((parserMap.GenerateGRM == false) && (parserMap.GenerateNull == true)) {
		// check sparse GRM file exists
		if _,err := os.Stat(parserMap.SparseGRM); err == nil {
				fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path and file are reachable for config variable SparseGRM.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable SparseGRM.\n", parserMap.SparseGRM)
				os.Exit(5)
			} else {
				fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable SparseGRM but the following error occurred: %v \n", err)
			}
		// check sampleID file exists
		if _,err := os.Stat(parserMap.SampleIDFile); err == nil {
				fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path and file are reachable for config variable SampleIDFile.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable SampleIDFile.\n", parserMap.SampleIDFile)
				os.Exit(5)
			} else {
				fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable SampleIDFile but the following error occurred: %v \n", err)
			}
	}


	// if using GenerateAssociations is set to true but reusing an existing Null model, then check these files exist
	if ((parserMap.GenerateNull == false) && (parserMap.GenerateAssociations == true)) {
		// check NullModelFile exists
		if _,err := os.Stat(parserMap.NullModelFile); err == nil {
				fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path and file are reachable for config variable NullModelFile.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable NullModelFile.\n", parserMap.NullModelFile)
				os.Exit(5)
			} else {
				fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable NullModelFile but the following error occurred: %v \n", err)
			}
		// check VarianceRatioFile exists
		if _,err := os.Stat(parserMap.VarianceRatioFile); err == nil {
				fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path and file are reachable for config variable VarianceRatioFile.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable VarianceRatioFile.\n", parserMap.VarianceRatioFile)
				os.Exit(5)
			} else {
				fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable SampleIDFile but the following error occurred: %v \n", err)
			}
	}

	// if using GenerateAssociations is set to true then check these files exist -- also check based on whether skipChunking is set to true/false
	if parserMap.GenerateAssociations == true {
		// check Impute directory exists
		if _,err := os.Stat(parserMap.ImputeDir); err == nil {
			fmt.Printf("[func(checkInput)] -- CONFIRMED! -- The path is reachable for config variable ImputeDir.\n")
		} else if os.IsNotExist(err) {
			fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable ImputeDir.\n", parserMap.ImputeDir)
			os.Exit(5)
		} else {
			fmt.Printf("[func(checkInput)] -- WARNING! -- File may exist from variable ImputeDir but the following error occurred: %v \n", err)
		}

		// check chromosome ranges are integers and separted by dash
		chroms := strings.Split(parserMap.Chromosomes, "-")
 		start := strings.TrimSpace(chroms[0])
 		end := strings.TrimSpace(chroms[1])

		checkStart,err:=  strconv.Atoi(start)
		if err != nil {
			fmt.Printf("[func(checkInput)] -- ERROR! -- There was a problem converting your starting chromosome.  Please be sure this in an integer between 1-22. You entered: %s.\n", start)
			os.Exit(5)
		}
		checkEnd,err := strconv.Atoi(end)
		if err != nil {
			fmt.Printf("[func(checkInput)] -- ERROR! -- There was a problem converting your ending chromosome.  Please be sure this in an integer between 1-22. You entered: %s.\n", end)
			os.Exit(5)
		}

		// check to make sure start-end is in ascending or equal order
		if checkStart <= checkEnd {
			fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Chromosome ranges are in proper order.\n")
		} else {
			fmt.Printf("[func(checkInput)] -- ERROR! -- The chromosome range must be in equal or ascending order.  Please rearrange.  You entered: %s.", parserMap.Chromosomes)
			os.Exit(5)
		}

		// check start chromosome is 1-22
		if ((checkStart < 1) || (checkStart > 22)) {
			fmt.Printf("[func(checkInput)] -- ERROR! -- The starting chromosome must be between 1-22. You entered: %d.\n", checkStart)
			os.Exit(5)
		}

		// check end chromosome is 1-22
		if ((checkEnd < 1) || (checkEnd > 22)) {
			fmt.Printf("[func(checkInput)] -- ERROR! -- The ending chromosome must be between 1-22. You entered: %d.\n", checkEnd)
			os.Exit(5)
		}
		
		// confirm this is string range
		if len(chroms) != 2 {
			fmt.Printf("[func(checkInput)] -- ERROR! -- Chromosomes must be an integer range.  Ex: 1-22, 1-1, 3-10, etc....  You entered: %s.\n", parserMap.Chromosomes)
			os.Exit(5)
		}

		// check imputation files and index files exist
		if parserMap.SkipChunking == false {
			var checkFileExists = make([]string, 0)
			for i:=checkStart; i<checkEnd+1; i++ {
				if parserMap.Build == "hg38" {
					chrom := strconv.Itoa(i)
					chrom = "chr"+chrom
					checkFileExists = append(checkFileExists, filepath.Join(parserMap.ImputeDir, chrom + parserMap.ImputeSuffix))
					checkFileExists = append(checkFileExists, filepath.Join(parserMap.ImputeDir, chrom + parserMap.ImputeSuffix + ".tbi"))
				} else {
					chrom := strconv.Itoa(i)
					checkFileExists = append(checkFileExists, filepath.Join(parserMap.ImputeDir,chrom + parserMap.ImputeSuffix))
					checkFileExists = append(checkFileExists, filepath.Join(parserMap.ImputeDir,chrom + parserMap.ImputeSuffix + ".tbi"))
				}
			}

			for _,val := range checkFileExists {
				if _,err := os.Stat(val); err == nil {
					fmt.Printf("[func(checkInput)] -- CONFIRMED! -- imputation file is reachable for config variable ImputeDir and ImputeSuffix.\n")
				} else if os.IsNotExist(err) {
					fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable ImputeDir and ImputeSuffix\n", val)
					os.Exit(5)
				} else {
					fmt.Printf("[func(checkInput)] -- WARNING! -- The file %s may exist from variable ImputeDir and ImputeSuffix but the following error occurred: %v \n", val, err)
				}					
			}

		} else if parserMap.SkipChunking == true {
			if _,err := os.Stat(parserMap.ChromosomeLengthFile); err == nil {
				fmt.Printf("[func(checkInput)] -- CONFIRMED! -- chromosome length file is reachable for config variable ChromosomeLengthFile.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable ChromosomeLengthFile.\n", parserMap.ChromosomeLengthFile)
				os.Exit(5)
			} else {
				fmt.Printf("[func(checkInput)] -- WARNING! -- The file %s may exist from variable ChromosomeLengthFile but the following error occurred: %v \n", parserMap.ChromosomeLengthFile, err)
			}

			// check if ImputationFileList exists that contains file chunk names
			if _,err := os.Stat(parserMap.ImputationFileList); err == nil {
				fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Imputation File List is reachable for config variable ImputationFileList.\n")
			} else if os.IsNotExist(err) {
				fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable ImputationFileList.\n", parserMap.ImputationFileList)
				os.Exit(5)
			} else {
				fmt.Printf("[func(checkInput)] -- WARNING! -- The file %s may exist from variable ChromosomeLengthFile but the following error occurred: %v \n", parserMap.ImputationFileList, err)
			}
		}

		// check this field is either DS or GT (parserMap.VcfField)
		if ((parserMap.VcfField == "DS") || (parserMap.VcfField == "GT")) {
			fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Config variable VcfField is set to %s.\n", parserMap.VcfField)
		} else {
			fmt.Printf("[func(checkInput)] -- ERROR! -- Please set VcfField as either True or False.  You entered: %s.\n", parserMap.VcfField)
			os.Exit(5)		
		}


		// check parserMap.IsDropMissingDosages is string-based true/false
		if ((parserMap.IsDropMissingDosages == "TRUE") || (parserMap.IsDropMissingDosages == "FALSE")) {
			fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Config variable IsDropMissingDosages is set to %s.\n", parserMap.IsDropMissingDosages)
		} else {
			fmt.Printf("[func(checkInput)] -- ERROR! -- Please set IsDropMissingDosages as either True or False.  You entered: %s.\n", parserMap.IsDropMissingDosages)
			os.Exit(5)		
		}
	}

	if parserMap.GenerateResults == true {
		// check if Association results file exists
		if _,err := os.Stat(parserMap.AssociationFile); err == nil {
			fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Association anaysis file is reachable for config variable AssociationFile.\n")
		} else if os.IsNotExist(err) {
			fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable AssociationFile.\n", parserMap.AssociationFile)
			os.Exit(5)
		} else {
			fmt.Printf("[func(checkInput)] -- WARNING! -- The file %s may exist from variable AssociationFile but the following error occurred: %v \n", parserMap.AssociationFile, err)
		}

		// check if info file exists
		if _,err := os.Stat(parserMap.InfoFile); err == nil {
			fmt.Printf("[func(checkInput)] -- CONFIRMED! -- Info file is reachable for config variable InfoFile.\n")
		} else if os.IsNotExist(err) {
			fmt.Printf("[func(checkInput)] -- ERROR! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable InfoFile.\n", parserMap.InfoFile)
			os.Exit(5)
		} else {
			fmt.Printf("[func(checkInput)] -- WARNING! -- The file %s may exist from variable AssociationFile but the following error occurred: %v \n", parserMap.InfoFile, err)
		}

		// 	
	}
}

func saveResults(bindPointTemp,outputPrefix,outDir string, saveChunks,saveTar bool) {
	save := time.Now()
	fmt.Printf("[func(saveResults)] -- UPDATE! -- begin transferring final results %s.\n", time.Now())
	err:=os.Mkdir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults"), 0755)
	if err != nil {
 		fmt.Printf("[func(saveResults) %s] -- ERROR! -- There was an error creating the final results directory. \t%v\n", time.Now(), err)
 		os.Exit(99)
 	}else {
 		fmt.Printf("[func(saveResults) %s] -- UPDATE! -- Created final results directory called %s\n", time.Now(), outputPrefix+"_finalResults")
 	}

 	_=os.Mkdir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults/grm_files"), 0755)
 	_=os.Mkdir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults/null_model_files"), 0755)
 	_=os.Mkdir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults/chunked_imputation_files"), 0755)
 	_=os.Mkdir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults/association_analysis_results"), 0755)
 	_=os.Mkdir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults/other"), 0755)


	matches := make([]string, 0)
    findThese := [14]string{"*.mtx.sampleIDs.txt", "*.sparseGRM.mtx", "*.sparseSigma.mtx", 
    			"*.varianceRatio.txt", "*.rda", "*.pdf", "*.png", "*_allChromosomeResultsMerged.txt", 
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
    	switch{
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".mtx.sampleIDs.txt") || strings.HasSuffix(fileName[len(fileName)-1], ".sparseGRM.mtx") || strings.HasSuffix(fileName[len(fileName)-1], ".sparseSigma.mtx")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/grm_files",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into grm_files %s] -- WARNING! -- Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".varianceRatio.txt") || strings.HasSuffix(fileName[len(fileName)-1], ".rda")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/null_model_files",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into null_model_files %s] -- WARNING! -- Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".png") || strings.HasSuffix(fileName[len(fileName)-1], ".pdf") || strings.HasSuffix(fileName[len(fileName)-1], "_allChromosomeResultsMerged.txt") || strings.HasSuffix(fileName[len(fileName)-1], ".txt.gz")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/association_analysis_results",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into association_analysis_results %s] -- WARNING! -- Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".vcf.gz") || strings.HasSuffix(fileName[len(fileName)-1], ".vcf.gz.tbi") || strings.HasSuffix(fileName[len(fileName)-1], "_chunkedImputationQueue.txt")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/chunked_imputation_files",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into association_analysis_results %s] -- WARNING! -- Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".log") || strings.HasSuffix(fileName[len(fileName)-1], ".err")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/other",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into other %s] -- WARNING! -- Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	}
    }

   
    if saveTar == true {
    	//tar final file
   		filesToTar, _ := ioutil.ReadDir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults"))
    	tarFileName, tarErr := os.Create(filepath.Join(bindPointTemp,outputPrefix+"_finalResults.tar"))
    	if tarErr != nil {
    		fmt.Printf("[func(saveResults) %s] -- WARNING! -- Error encountered when creating compressed tar file \t%v\n", time.Now(),tarErr)
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
      			fmt.Printf("[func(saveResults) -- transferring final results %s] -- WARNING! -- Problem writing file to tar.  The following error was encountered: %v\n",time.Now(),err)
      		}
 
	      	_, err = io.Copy(tarFileWriter, file)
   	   		if err != nil {
      			fmt.Printf("[func(saveResults) -- transferring final results %s] -- WARNING! -- Problem copying file to tar.  The following error was encountered: %v\n",time.Now(),err)
      		} 
    	}
	
    	// check if tmp bindpoint is different from outdir; if true then move to outdir, if false keep in tmpDir
    	tmpLoc := strings.TrimSpace(strings.TrimSuffix(bindPointTemp, "/"))
		finalLoc := strings.TrimSpace(strings.TrimSuffix(outDir, "/"))

    	if tmpLoc  != finalLoc {
    		os.Rename(filepath.Join(bindPointTemp,outputPrefix+"_finalResults.tar"), filepath.Join(outDir,outputPrefix+"_finalResults.tar"))
    	}
    	os.RemoveAll(filepath.Join(bindPointTemp,outputPrefix+"_finalResults"))
    }
    fmt.Printf("[func(saveResults)] -- UPDATE! -- finished transferring final results %s. Time Elapsed: %.2f minutes\n", time.Now(), time.Since(save).Minutes())
}

func saveQueue (queueFile string, f *os.File) {
	queueFileSave.Lock()
	_, err := f.WriteString(queueFile+"\n")
	if err != nil {
		fmt.Printf("[func(saveQueue) %s] -- WARNING! -- There was an error when writing %s queue to file:\t%v\n", time.Now(),queueFile, err)
		queueFileSave.Unlock()

	}else{
		fmt.Printf("[func(saveQueue) %s] -- UPDATE! -- Saved chunked file to queue list: %s\n", time.Now(),queueFile)
		f.Sync()
		queueFileSave.Unlock()
	}
}

func usePrevChunks (start,end,build,imputeDir,imputationFileList string) {
	defer wgAllChunks.Done()

	var chromsToProcess = make([]string, 0)

	startInt,_ :=  strconv.Atoi(start)
	endInt,_ := strconv.Atoi(end)

	for i:=startInt; i < endInt+1; i++ {
		intervals := strconv.Itoa(i)
		if build == "hg38" {
			chromsToProcess = append(chromsToProcess, "chr"+ intervals+"_")
		} else if build == "hg19" {
			chromsToProcess = append(chromsToProcess, intervals+"_")
		} else {
			fmt.Printf("[func(usePrevChunks)] -- ERROR! -- There was an error with the human genome build specifed.  You entered %v.  Please select either hg38 or hg19.\n", build)
			os.Exit(17)
		}
	}

	fileQueue, err := os.Open(imputationFileList)
	if err != nil {
		fmt.Printf("[func(usePrevChunks) %s] -- ERROR! -- There was an error opening the imputation chunk file list. The error is as follows: %v\n", time.Now(),err)
		os.Exit(17)
	}

	scanner := bufio.NewScanner(fileQueue)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		for _,chromList := range chromsToProcess {
			if strings.HasPrefix(scanner.Text(), chromList) {
				if _,err := os.Stat(filepath.Join(parserMap.ImputeDir, scanner.Text())); err == nil {
					changeQueueSize.Lock()
					processQueue = append(processQueue, scanner.Text())
					changeQueueSize.Unlock()
					fmt.Printf("[func(usePrevChunks) %s] -- UPDATE! -- %s, has been added to the processing queue.\n", time.Now(), scanner.Text())
				} else if os.IsNotExist(err) {
					fmt.Printf("[func(usePrevChunks) %s] %s, -- WARNING! -- Ooops, the path %s does not exist!  Please confirm this path and file are reachable.  Skipping this file...\n", time.Now(), filepath.Join(parserMap.ImputeDir,scanner.Text()))
				} else {
					fmt.Printf("[func(usePrevChunks) %s] %s, -- WARNING! -- The file %s may exist from variable ImputeDir and ImputeSuffix but the following error occurred: %v.  Skipping this file...\n", time.Now(), filepath.Join(parserMap.ImputeDir, scanner.Text()), err)
				}
				break
			} else {
				continue
			}
		}
	}

	allChunksFinished--
}

func findElement (headerSlice []string, element string) {
	for _,headerVal := range headerSlice {
		if headerVal == element {
			fmt.Printf("[func(findElement)] -- CONFIRMED! -- %v in phenoFile\n", element)
			return
		}
	}
	fmt.Printf("[func(findElement)] -- ERROR! -- %v is not listed in phenofile.  Please update phenofile.\n", element)
	os.Exit(20)
}

func parser (configFile string) {
	fileBytes, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("[func(parser)] -- ERROR! -- There was a problems reading config file.\n")
		os.Exit(3)
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
				fmt.Printf("[func(parser)] -- ERROR! -- There was an error converting ChunkVariants to integer: %v\n", err)
				os.Exit(3)
			}
			parserMap.ChunkVariants = chunkSize
		case strings.TrimSpace(tmpParse[0]) == "SaveChunks":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.SaveChunks = false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.SaveChunks = true
			} else {
				fmt.Printf("[func(parser)] -- ERROR! -- %v is not a valid option for SaveChunks.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(3)
			}
		case strings.TrimSpace(tmpParse[0]) == "SkipChunking":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.SkipChunking = false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.SkipChunking = true
			} else {
				fmt.Printf("[func(parser)] -- ERROR! -- %v is not a valid option for SkipChunking.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(3)
			}
		case strings.TrimSpace(tmpParse[0]) == "GenerateGRM":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.GenerateGRM = false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.GenerateGRM = true
			} else {
				fmt.Printf("[func(parser)] -- ERROR! -- %v is not a valid option for GenerateGRM.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(3)
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
			parserMap.VcfField= strings.ToUpper(strings.TrimSpace(tmpParse[1]))
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
		case strings.TrimSpace(tmpParse[0]) == "SaveAsTar":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.SaveAsTar= false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.SaveAsTar= true
			} else {
				fmt.Printf("[func(parser)] -- ERROR! -- %v is not a valid option for SaveAsTar.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(3)
			}
		case strings.TrimSpace(tmpParse[0]) == "GenerateNull":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.GenerateNull= false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.GenerateNull= true
			} else {
				fmt.Printf("[func(parser)] -- ERROR! -- %v is not a valid option for GenerateNull.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(3)
			}
		case strings.TrimSpace(tmpParse[0]) == "GenerateAssociations":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.GenerateAssociations= false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.GenerateAssociations= true
			} else {
				fmt.Printf("[func(parser)] -- ERROR! -- %v is not a valid option for GenerateAssociations.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(3)
			}
		case strings.TrimSpace(tmpParse[0]) == "GenerateResults":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.GenerateResults= false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.GenerateResults= true
			} else {
				fmt.Printf("[func(parser)] -- ERROR! -- %v is not a valid option for GenerateAssociations.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(3)
			}
		case strings.TrimSpace(tmpParse[0]) == "NullModelFile":
			parserMap.NullModelFile = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "VarianceRatioFile":
			parserMap.VarianceRatioFile = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "AssociationFile":
			parserMap.AssociationFile = strings.TrimSpace(tmpParse[1])
		}
		if err == io.EOF {
			fmt.Println("[func(parser)] -- UPDATE! -- Finished parsing config file!\n")
			break
		}
	}
}