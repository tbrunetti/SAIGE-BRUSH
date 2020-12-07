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
	fmt.Printf("[func(main) Thread Allocation %s] %v total CPUs available.\n", time.Now(),totalCPUsAvail)
	if parserMap.NThreads == "" {
		parserMap.NThreads = strconv.Itoa(totalCPUsAvail)
		runtime.GOMAXPROCS(totalCPUsAvail)
	}else {
		maxThreads,err := strconv.Atoi(parserMap.NThreads)
		if err != nil{
			fmt.Printf("[func(main) Thread Allocation %s] There was a problem allocating your threads.  You entered: %v\n, %v", time.Now(),parserMap.NThreads, err)
			os.Exit(42)
		}
		runtime.GOMAXPROCS(maxThreads)
	}

	fmt.Printf("[func(main) Thread Allocation %s] %v total CPUs will be used.\n", time.Now(), parserMap.NThreads)
	
	
	
	allChunksFinished = 1

	// Before starting pipeline perform a basic input check
	checkInput(parserMap.MAC,parserMap.MAF,parserMap.PhenoFile,parserMap.Pheno,parserMap.Covars,parserMap.SampleID)

	 // create tmp folder to be deleted at end of run
 	err:=os.Mkdir(filepath.Join(parserMap.BindPointTemp, "tmp_saige"), 0755)
 	if err != nil {
 		fmt.Printf("[func(main) %s] There was an error creating the tmp directory. \t%v\n", time.Now(), err)
 		os.Exit(42)
 	}else {
 		fmt.Printf("[func(main) %s] Created tmp directory called tmp_saige in %s\n", time.Now(), parserMap.BindPointTemp)
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


		fmt.Printf("[func(main) generate GRM %s] There are a total of %v snps that meet the maf requirements for GRM calculation.\n", time.Now(),totalSNPs[0])


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
		fmt.Printf("[func(main) -- generate GRM %s] Sparse GRM path located at: %s\n", time.Now(),parserMap.SparseGRM)
		parserMap.SampleIDFile = filepath.Join(parserMap.BindPointTemp, "tmp_saige", parserMap.OutPrefix+"_relatednessCutoff_"+parserMap.Rel+"_"+string(totalSNPs[0])+"_randomMarkersUsed.sparseGRM.mtx.sampleIDs.txt")
		fmt.Printf("[func(main) -- generate GRM %s] Sparse GRM sampleID path located at: %s\n", time.Now(),parserMap.SampleIDFile)
	} else {
		if _,err := os.Stat(parserMap.SparseGRM); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("[func(main) -- skip GRM; use supplied check] ERROR! Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable SparseGRM.\n", parserMap.SparseGRM)
				os.Exit(42)
			}else {
				fmt.Printf("[func(main) -- skip GRM; use supplied check] CONFIRMED! The path and file are reachable for config variable SparseGRM.\n", parserMap.SparseGRM)
			}
		}
		if _,err := os.Stat(parserMap.SampleIDFile); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("[func(main) -- skip GRM; use supplied check] ERROR! Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable SampleIDFile.\n", parserMap.SampleIDFile)
				os.Exit(42)
			}
		}else {
				fmt.Printf("[func(main) -- skip GRM; use supplied check] CONFIRMED! The path and file are reachable for config variable SampleIDFile.\n", parserMap.SampleIDFile)

		}		
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
				fmt.Printf("[func(main) null thread allocation %s] There was an error converting threads: %v\n", time.Now(),err)
				os.Exit(42)
			}
			toNull := math.Ceil(float64(threadsNull) * 0.75)
			toChunk := math.Ceil(float64(threadsNull) - toNull)
			toNullString := fmt.Sprintf("%f", toNull)
			
			fmt.Printf("func(main) null thread allocation %s] There are %v threads requested.  %v are reserverd for the null model generation. %v are reserved for chunking.\n", time.Now(), threadsNull, toNull, toChunk)
			
			go nullModel(parserMap.BindPoint,parserMap.BindPointTemp,parserMap.Container,parserMap.SparseGRM,parserMap.SampleIDFile,parserMap.PhenoFile,parserMap.Plink,
				parserMap.Trait,parserMap.Pheno,parserMap.InvNorm,parserMap.Covars,parserMap.SampleID,toNullString,parserMap.SparseKin,parserMap.Markers,parserMap.OutDir,
				parserMap.OutPrefix,parserMap.Rel,parserMap.Loco,parserMap.CovTransform) 
			time.Sleep(1* time.Minute) 

		}
	}else if ((parserMap.GenerateNull == false) && (parserMap.GenerateAssociations==true)) {
		if _,err := os.Stat(parserMap.NullModelFile); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("[func(main) -- skip null model file; use supplied check] ERROR! Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable NullModelFile.\n", parserMap.NullModelFile)
				os.Exit(42)
			}else {
				fmt.Printf("[func(main) -- skip null model file; use supplied check] CONFIRMED! The path and file are reachable for config variable NullModelFile.\n", parserMap.NullModelFile)
			}
		}
		if _,err := os.Stat(parserMap.VarianceRatioFile); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("[func(main) -- skip variance ratio calculation; use supplied check] ERROR! Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable VarianceRatioFile.\n", parserMap.VarianceRatioFile)
				os.Exit(42)
			}else {
				fmt.Printf("[func(main) -- skip variance ratio calculation; use supplied check] CONFIRMED! The path and file are reachable for config variable VarianceRatioFile.\n", parserMap.VarianceRatioFile)
			}
		}
	}else {
		fmt.Printf("[func(main)] null model not needed since no association analyses will be calculated.\n")
	}
 	
 	
 	/* STEP1a: Chunks Queue or Use of previously chunked imputation data
 	chunk can run the same time as the null model; and the association analysis needs to wait for the 
 	null to finish and chunk to finish*/
 	if ((parserMap.SkipChunking ==  false) && (parserMap.GenerateAssociations == true)) {
 		fmt.Printf("[func(main) Queue status] Chunking files for queue for association analyses\n")
 		wgAllChunks.Add(1)
    	go chunk(start,end,parserMap.Build,parserMap.OutDir,parserMap.ChromosomeLengthFile,parserMap.ImputeDir,parserMap.ImputeSuffix,parserMap.BindPoint,
    		parserMap.BindPointTemp,parserMap.Container,parserMap.ChunkVariants,f)
 	}else if ((parserMap.SkipChunking ==  true) && (parserMap.GenerateAssociations == true)) {
 		fmt.Printf("[func(main) Queue status] Reusing previously chunked and index files for queue for association analyses\n")
 		wgAllChunks.Add(1)
 		go usePrevChunks(parserMap.ImputeDir, parserMap.ImputationFileList)
 	}else {
 		fmt.Printf("[func(main) Queue status] Skipping queue since no associations are required\n")
 	}
  
    
    // wait for null model to finish before proceeding with association analysis -- no need to wait for chunk to finish; 
    // if GenerateNull is set to false, wgNull sync will already be 0 since no jobs addded so will move past this command
    wgNull.Wait()

	
    /* STEP2: Association Analysis Queue
	while loop to keep submitting jobs until queue is empty and no more subsets are available*/
	if parserMap.GenerateAssociations == true {
		for allChunksFinished == 1 || len(processQueue) != 0 {
			if ((allAssocationsRunning < totalCPUsAvail) && (len(processQueue) > 0)) {
				fmt.Printf("[func(main) %s] The total number of associations running is %v.\n", time.Now(), allAssocationsRunning)
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
	    
	    fmt.Printf("[func(main) -- concatenate %s] Finished all association results. Time Elapsed: %.2f minutes\n", time.Now(),time.Since(concatTime).Minutes())
	}
    
    // STEP3: Clean, Visualize, and Summarize results
    if parserMap.GenerateResults == true {
    	if parserMap.GenerateAssociations == false {
			if _,err := os.Stat(parserMap.AssociationFile); err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("[func(main) -- skip associationAnalysis; use supplied check] ERROR! Ooops, the path %s does not exist!  Please confirm this path and file are reachable for config variable AssociationFile.\n", parserMap.AssociationFile)
					os.Exit(42)
				}else {
					fmt.Printf("[func(main) -- skip associationAnalysis; use supplied check] CONFIRMED! The path and file are reachable for config variable AssociationFile.\n", parserMap.AssociationFile)
				}
			}
    	}else{
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


	    fmt.Printf("[func(main) -- clean and graph results %s] Finished all data clean up, visualizations, and summarization. Time Elapsed: %.2f minutes\n", time.Now(), time.Since(graph).Minutes())
	}

    //TODO: clean up temp files
    saveResults(parserMap.BindPointTemp,parserMap.OutPrefix,parserMap.OutDir,parserMap.SaveChunks,parserMap.SaveAsTar)

    // Pipeline Finish
    fmt.Printf("[func(main) %s] All threads are finished and pipeline is complete!\n", time.Now())
}




func chunk(start,end,build,outDir,chromosomeLengthFile,imputeDir,imputeSuffix,bindPoint,bindPointTemp,container string, chunkVariants int, f *os.File) {
	defer wgAllChunks.Done() //once function finishes decrement sync object

	fileBytes, err := os.Open(chromosomeLengthFile)
	if err != nil {
		fmt.Printf("func(chunk) %s] There was a problems reading in the chromosome length file.", time.Now())
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
		fmt.Printf("[func(smallerChunk) %s] %s overall: Error in total variants call:\n%v", formatted,chrom, err)
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
		wgSmallChunk.Wait()
	}
	
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
	bcfToolsErr := subset.Run()
	if bcfToolsErr != nil {
		fmt.Printf("[func(processing) %s] BcfTools subsetting of %s finished with error: %v\n", time.Now(), chrom+":"+lowerValStr+"-"+upperValStr, bcfToolsErr)
		return
	}

	index := exec.Command("singularity", "run", "-B", bindPoint+","+bindPointTemp, container, "/opt/tabix",
		"-p",
		"vcf",
		filepath.Join(bindPointTemp,"tmp_saige",chrom+"_")+loopNum+"_"+lowerValStr+"_"+upperValStr+"_"+imputeSuffix)
	indexErr:=index.Run()
	if indexErr != nil {
		fmt.Printf("[func(processing) %s] Indexing of %s finished with error: %v\n", time.Now(), chrom+":"+lowerValStr+"-"+upperValStr, indexErr)
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
		

	default:
		fmt.Printf("func(nullModel) %s] Please select trait type as either binary or quantitative.  You entered: %s.\n", time.Now(), trait)
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
		fmt.Printf("[func(checkInput)] There was an error opening your phenotype file: \n\t%v", err)
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


func saveResults(bindPointTemp,outputPrefix,outDir string, saveChunks,saveTar bool) {
	save := time.Now()
	fmt.Printf("[func(saveResults) -- begin transferring final results %s]\n", time.Now())
	err:=os.Mkdir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults"), 0755)
	if err != nil {
 		fmt.Printf("[func(saveResults) %s] There was an error creating the final results directory. \t%v\n", time.Now(), err)
 		os.Exit(42)
 	}else {
 		fmt.Printf("[func(saveResults) %s]Created final results directory called %s\n", time.Now(), outputPrefix+"_finalResults")
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
    			fmt.Printf("[func(saveResults) -- transferring final results into grm_files %s] Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".varianceRatio.txt") || strings.HasSuffix(fileName[len(fileName)-1], ".rda")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/null_model_files",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into null_model_files %s] Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".png") || strings.HasSuffix(fileName[len(fileName)-1], ".pdf") || strings.HasSuffix(fileName[len(fileName)-1], "_allChromosomeResultsMerged.txt") || strings.HasSuffix(fileName[len(fileName)-1], ".txt.gz")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/association_analysis_results",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into association_analysis_results %s] Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".vcf.gz") || strings.HasSuffix(fileName[len(fileName)-1], ".vcf.gz.tbi") || strings.HasSuffix(fileName[len(fileName)-1], "_chunkedImputationQueue.txt")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/chunked_imputation_files",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into association_analysis_results %s] Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	case (strings.HasSuffix(fileName[len(fileName)-1], ".log") || strings.HasSuffix(fileName[len(fileName)-1], ".err")):
    		err := os.Rename(fileTransfer, filepath.Join(bindPointTemp,outputPrefix+"_finalResults/other",fileName[len(fileName)-1]))
    		if err != nil {
    			fmt.Printf("[func(saveResults) -- transferring final results into other %s] Problem transferring file %s to %s.\n\tThe following error was encountered: %v\n", time.Now(),filepath.Join(bindPointTemp,fileTransfer),filepath.Join(outDir,fileTransfer),err)
    		}
    	}
    }

   
    if saveTar == true {
    	//tar final file
   		filesToTar, _ := ioutil.ReadDir(filepath.Join(bindPointTemp,outputPrefix+"_finalResults"))
    	tarFileName, tarErr := os.Create(filepath.Join(bindPointTemp,outputPrefix+"_finalResults.tar"))
    	if tarErr != nil {
    		fmt.Printf("[func(saveResults) %s] Error encountered when creating compressed tar file \t%v\n", time.Now(),tarErr)
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
      			fmt.Printf("[func(saveResults) -- transferring final results %s] Problem writing file to tar.  The following error was encountered: %v\n",time.Now(),err)
      		}
 
	      	_, err = io.Copy(tarFileWriter, file)
   	   		if err != nil {
      			fmt.Printf("[func(saveResults) -- transferring final results %s] Problem copying file to tar.  The following error was encountered: %v\n",time.Now(),err)
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
    fmt.Printf("[func(saveResults) -- finished transferring final results %s] Time Elapsed: %.2f minutes\n", time.Now(), time.Since(save).Minutes())
}


func saveQueue (queueFile string, f *os.File) {
	queueFileSave.Lock()
	_, err := f.WriteString(queueFile+"\n")
	if err != nil {
		fmt.Printf("[func(saveQueue) %s] There was an error when writing %s queue to file:\t%v\n", time.Now(),queueFile, err)
		queueFileSave.Unlock()

	}else{
		fmt.Printf("[func(saveQueue) %s] Saved chunked file to queue list: %s\n", time.Now(),queueFile)
		f.Sync()
		queueFileSave.Unlock()
	}
}


func usePrevChunks (imputeDir,imputationFileList string) {
	defer wgAllChunks.Done()
	fileQueue, err := os.Open(imputationFileList)
	if err != nil {
		fmt.Printf("[func(usePrevChunks) %s] There was an error opening the imputation chunk file list. The error is as follows: %v\n", time.Now(),err)
		os.Exit(42)
	}

	scanner := bufio.NewScanner(fileQueue)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		changeQueueSize.Lock()
		processQueue = append(processQueue, scanner.Text())
		changeQueueSize.Unlock()
		fmt.Printf("[func(usePrevChunks) %s] %s, has been added to the processing queue.\n", time.Now(), scanner.Text())
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
		fmt.Printf("[func(parser)] There was a problems reading config file.\n")
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
		case strings.TrimSpace(tmpParse[0]) == "SaveAsTar":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.SaveAsTar= false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.SaveAsTar= true
			} else {
				fmt.Printf("[func(parser)] %v is not a valid option for SaveAsTar.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(42)
			}
		case strings.TrimSpace(tmpParse[0]) == "GenerateNull":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.GenerateNull= false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.GenerateNull= true
			} else {
				fmt.Printf("[func(parser)] %v is not a valid option for GenerateNull.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(42)
			}
		case strings.TrimSpace(tmpParse[0]) == "GenerateAssociations":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.GenerateAssociations= false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.GenerateAssociations= true
			} else {
				fmt.Printf("[func(parser)] %v is not a valid option for GenerateAssociations.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(42)
			}
		case strings.TrimSpace(tmpParse[0]) == "GenerateResults":
			if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "FALSE" {
				parserMap.GenerateResults= false
			} else if strings.ToUpper(strings.TrimSpace(tmpParse[1])) == "TRUE" {
				parserMap.GenerateResults= true
			} else {
				fmt.Printf("[func(parser)] %v is not a valid option for GenerateAssociations.\n", strings.TrimSpace(tmpParse[1]))
				os.Exit(42)
			}
		case strings.TrimSpace(tmpParse[0]) == "NullModelFile":
			parserMap.NullModelFile = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "VarianceRatioFile":
			parserMap.VarianceRatioFile = strings.TrimSpace(tmpParse[1])
		case strings.TrimSpace(tmpParse[0]) == "AssociationFile":
			parserMap.AssociationFile = strings.TrimSpace(tmpParse[1])
		}
		if err == io.EOF {
			fmt.Println("[func(parser)] Finished parsing config file!\n")
			break
		}
	}
}