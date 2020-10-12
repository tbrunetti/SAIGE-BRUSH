
binary = TRUE
pheno = "afib"

formatPheno <- function(genesisObj){
	load(GENESISdataObj)
	genPCs <- as.data.frame(mypcair$vectors)
	totalPCs <- length(genPCs)
	pcNames <- list()
	for (i in seq(1, totalPCs)){
		pcNames <- c(pcNames, paste("pc", i, sep=""))
	}
	names(genPCs) <- pcNames
	genPCs$genesisIndex <- rownames(genPCs)
	genPCs$SAIGEindex <- as.numeric(genPCs$genesisIndex) - 1

	genKey <- read.delim("KC_BCP_final_fam_to_update.txt", header=F, sep="\t")
	names(genKey) <- c("IID", "genesisIndex")
	combinePCandIDs <- merge(genPCs, genKey, by="genesisIndex", all=T)
	if (dim(combinePCandIDs)[1] == dim(genPCs)[1]){
		print("All GENESIS pc IDs are 1 to 1 matched with GENESIS key file")
	}else{
		stop("ERROR! PC vector and GENESIS key file have mismatches. Please check files. Exiting.")
	}

	phenoCovFile = read.delim("Wiley_afib.txt", header=T, sep="\t") # IID header required, and any phenotypes and covariates
	finalFile <- merge(x= phenoCovFile, y = combinePCandIDs, by="IID", all=T)
	if (dim(finalFile)[1] == dim(combinePCandIDs)[1]){
		print("Confirmed phenotype/covariate file dimensions match full GENESIS PC matrix")
	}else{
		stop("ERROR! phenotype/covariate file dimensions are mismatched with full GENESIS PC matrix. Please check files. Exiting.")
	}

	# check binary traits - Needs to be 0 or 1 for SAIGE
	names(table(finalFile[pheno])) # lists all pheno factors


	write.table(finalFile, "myOutputFile.txt", sep="\t", row.names=F, quote=F)


}

