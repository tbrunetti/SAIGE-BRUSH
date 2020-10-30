#prepare info file
#IMPUTED = imputed only
#TYPED = imputed and typed
#TYPED-ONLY = typed only
#library(readr)
#infoFile <- fread("SAIGE/extdataCustom/allAutosomes.rsq70.info.txt", header = T)
#infoFile$GENOTYPE_STATUS[which(infoFile$IMPUTED == 1)] <- "imputed"
#infoFile$GENOTYPE_STATUS[which(infoFile$TYPED_ONLY == 1)] <- "genotyped"
#infoFile$GENOTYPE_STATUS[which(infoFile$TYPED == 1)] <- "imputed and genotyped" 
#infoFile <- infoFile[,c("CHROM", "POS", "REF", "ALT", "R2", "ER2", "GENOTYPE_STATUS")]
#names(infoFile)<- c("CHR", "POS", "Allele1", "Allele2", "R2", "ER2", "GENOTYPE_STATUS")
#write_tsv(infoFile, "allAutosomes.rs70.info.SAIGE.txt", append = FALSE, col_names = TRUE)


cleanAndGraph <- function(assocFile, infoFile, dataOutputPrefix, pheno, covars, macFilter, mafFilter, traitType, nThreads){
  library(qqman)
  library(data.table)
  library(readr)
  library(plyr)
  library(png)
  library(gridExtra)
  library(ggplot2)
  library(grid)
  library(ggplotify)
  
  assocFile="~/SAIGE/extdataCustom/ADMIRE_AD_CCPMbb_freeze_v1.3_chrm21_associationAnalysis_results.txt"
  infoFile="~/allAutosomes.rs70.info.SAIGE.txt"
  dataOutputPrefix="test_new_graph"
  macFilter="10"
  mafFilter="0.05"
  traitType="binary"
  nThreads=3
  covars="PC1,PC2,PC3,PC4,PC5,SAIGE_GENDER,age"
  pheno="multiple_sclerosis"
  # read in concatenated SAIGE association results across all 
  autosomalAssocResults <- fread(assocFile, header=T, showProgress = T, nThread = nThreads)
  
  # merge info fields
  infoFields <- fread(infoFile , header = T, showProgress = T, nThread = nThreads)
  
  # perform a left only merge on info fields as long as all 4 columns match and only use the first match
  autosomalAssocResults <- join(x=autosomalAssocResults, y=infoFields, match = "first", type="left", by=c("CHR", "POS", "Allele1", "Allele2"))
  # by dropping "chr" it makes it compatible for querying searches numerically and easier for Manhattan plot functions
  autosomalAssocResults$CHR <- gsub(pattern = "chr", replacement = "", ignore.case = FALSE, x = autosomalAssocResults$CHR)
  # change types and make a new MAF column to confirm strand flip switches
  autosomalAssocResults$CHR <- as.numeric(autosomalAssocResults$CHR)
  autosomalAssocResults$MAF <- as.numeric(autosomalAssocResults$AF_Allele2)
  
  # allele2 is not always the minor allele -- therefore, check if minor and if not (1 - AF_Allele2) and flip beta
  # create mahor and minor allele columns which are more descriptive than allele1 and allele2 for users
  snpsThatNeedFlipping <- which(as.numeric(autosomalAssocResults$AF_Allele2) > 0.50)
  autosomalAssocResults$majorAllele <- as.character(autosomalAssocResults$Allele1)
  autosomalAssocResults$minorAllele <- as.character(autosomalAssocResults$Allele2)
  autosomalAssocResults$majorAllele[snpsThatNeedFlipping] <- as.character(autosomalAssocResults$Allele2)[snpsThatNeedFlipping]
  autosomalAssocResults$minorAllele[snpsThatNeedFlipping] <- as.character(autosomalAssocResults$Allele1)[snpsThatNeedFlipping]
  
  # note: BETA_MAF is the new BETA value after strand flipping and then calculate minor allele count using corrected MAF
  autosomalAssocResults$MAF[which(as.numeric(autosomalAssocResults$AF_Allele2) > 0.50)]<-  1-as.numeric(autosomalAssocResults$MAF[which(as.numeric(autosomalAssocResults$AF_Allele2) > 0.50)])
  autosomalAssocResults$BETA_MAF <- as.numeric(autosomalAssocResults$BETA)
  autosomalAssocResults[which(as.numeric(autosomalAssocResults$AF_Allele2) > 0.50), "BETA_MAF"] <- as.numeric(autosomalAssocResults$BETA[which(as.numeric(autosomalAssocResults$AF_Allele2) > 0.50)]) * (-1)
  autosomalAssocResults$MAC <- 2 * as.numeric(autosomalAssocResults$MAF) * as.numeric(autosomalAssocResults$N)

  # calculate odds ratio, log(odds ratio), -log10(pvalue), lower and upper 95% CI for odds ratio for GBE
  autosomalAssocResults$OR <- exp(as.numeric(autosomalAssocResults$BETA_MAF))
  autosomalAssocResults$LogOR <- log(autosomalAssocResults$OR)
  autosomalAssocResults$negLog10pvalue <- log10(as.numeric(autosomalAssocResults$p.value)) * -1
  autosomalAssocResults$Lower95OR <- exp(as.numeric(autosomalAssocResults$BETA_MAF) - 1.96*as.numeric(autosomalAssocResults$SE))
  autosomalAssocResults$Upper95OR <- exp(as.numeric(autosomalAssocResults$BETA_MAF) + 1.96*as.numeric(autosomalAssocResults$SE))
  
  # if the trait being test is binary then run the following if statement
  if (traitType == 'binary'){
    # make a new column to base allele dominance/recessive counts to reflect the minor allele instead of allele2
    autosomalAssocResults$casesHomozygousMinor <- autosomalAssocResults$homN_Allele2_cases
    autosomalAssocResults$casesHeterozygous <- autosomalAssocResults$hetN_Allele2_cases
    autosomalAssocResults$controlHomozygousMinor <- autosomalAssocResults$homN_Allele2_ctrls
    autosomalAssocResults$controlHeterozygous <- autosomalAssocResults$hetN_Allele2_ctrls
   
    # convert the homozygous allele2 count values into homozygous minor allele values for both cases and controls for alleles that needed to be flipped
    autosomalAssocResults$casesHomozygousMinor[snpsThatNeedFlipping] <- as.numeric(autosomalAssocResults$N.Cases)[snpsThatNeedFlipping] -
      (as.numeric(autosomalAssocResults$homN_Allele2_cases)[snpsThatNeedFlipping] + 
         as.numeric(autosomalAssocResults$hetN_Allele2_cases)[snpsThatNeedFlipping])
    
    autosomalAssocResults$controlHomozygousMinor[snpsThatNeedFlipping] <- as.numeric(autosomalAssocResults$N.Controls)[snpsThatNeedFlipping] -
      (as.numeric(autosomalAssocResults$homN_Allele2_ctrls)[snpsThatNeedFlipping] + 
         as.numeric(autosomalAssocResults$hetN_Allele2_ctrls)[snpsThatNeedFlipping])
    
    # to ensure confidentiality any counts of dom/recessive alleles greater than 0 but less then 10 are masked and generalized to "<10"
    autosomalAssocResults$casesHomozygousMinor[which((autosomalAssocResults$casesHomozygousMinor<10) & (autosomalAssocResults$casesHomozygousMinor > 0))] <- "<10"
    autosomalAssocResults$casesHeterozygous[which((autosomalAssocResults$casesHeterozygous<10) & (autosomalAssocResults$casesHeterozygous > 0))] <- "<10"
    autosomalAssocResults$controlHomozygousMinor[which((autosomalAssocResults$controlHomozygousMinor<10) & (autosomalAssocResults$controlHomozygousMinor > 0))] <- "<10"
    autosomalAssocResults$controlHeterozygous[which((autosomalAssocResults$controlHeterozygous<10) & (autosomalAssocResults$controlHeterozygous > 0))] <- "<10"
    
    #select and subset the table to only interested columns
    columnSubset <- c("CHR", "POS", "majorAllele", "minorAllele", "SNPID", "BETA_MAF", "SE", "OR", "LogOR", "Lower95OR", "Upper95OR", 
                      "MAF", "MAC", "p.value", "N", "N.Cases", "N.Controls", "casesHomozygousMinor", "casesHeterozygous", 
                      "controlHomozygousMinor", "controlHeterozygous", "negLog10pvalue",  "R2", "ER2", "GENOTYPE_STATUS")
    
    autosomalAssocResults <- autosomalAssocResults[,..columnSubset]
      names(autosomalAssocResults) <- c("CHR", "POS", "majorAllele", "minorAllele", "SNPID", "BETA", "SE", "OR", "LogOR", "Lower95OR", "Upper95OR", 
                                        "MAF", "MAC", "p.value", "N", "N.Cases", "N.Controls", "casesHomMinor", "casesHet", 
                                        "controlHomMinor", "controlHet", "negLog10pvalue","R2", "ER2", "GENOTYPE_STATUS")
    
    # write a copy of unfilterd table
    fwrite(autosomalAssocResults, file = paste(dataOutputPrefix, "GWASresults_allSNPs_noFiltering.txt.gz", sep="_"), append=FALSE, col.names=TRUE, showProgress = T, nThread = nThreads, compress = "gzip", sep = "\t")
    #write_tsv(autosomalAssocResults, paste(dataOutputPrefix, "GWASresults_allSNPs_noFiltering.txt", sep="_"), append=FALSE, col_names=TRUE)
    
    # filter by MAC and MAF based on user input to generate filtered common variants file (>macFilter, >mafFilter)
    commonClean <- autosomalAssocResults[which(as.numeric(autosomalAssocResults$MAC) > macFilter & 
                                                 as.numeric(autosomalAssocResults$MAF) > mafFilter),]
    fwrite(commonClean, file = paste(dataOutputPrefix, "GWASresults_commonSNPs_cleaned.txt.gz", sep="_"), append=FALSE, col.names = T, showProgress = T, nThread = nThreads, compress = "gzip", sep = "\t")
    #write_tsv(commonClean, paste(dataOutputPrefix, "GWASresults_commonSNPs_cleaned.txt", sep="_"), append=FALSE, col_names=TRUE)

    # filter by MAC and MAF based on user input to generate filtered rare variants file (>macFilter, <=mafFilter)
    rareClean <- autosomalAssocResults[which(as.numeric(autosomalAssocResults$MAC) > macFilter & 
                                               as.numeric(autosomalAssocResults$MAF) <= mafFilter),]
    fwrite(rareClean, file = paste(dataOutputPrefix, "GWASresults_rareSNPs_cleaned.txt.gz", sep="_"), append=FALSE, col.names=TRUE, showProgress = T, nThread = nThreads, compress = "gzip", sep = "\t")
    #write_tsv(rareClean, paste(dataOutputPrefix, "GWASresults_rareSNPs_cleaned.txt", sep="_"), append=FALSE, col_names=TRUE)
  }
  
  
  # if the trait being test is quantitative then run the following if statement
  if (traitType == 'quantitative'){
    #select and subset the table to only interested columns
    columnSubset <- c("CHR", "POS", "majorAllele", "minorAllele", "SNPID", "BETA_MAF", "SE", "OR", "LogOR", "Lower95OR", "Upper95OR", 
                      "MAF", "MAC", "p.value", "N", "negLog10pvalue",  "R2", "ER2", "GENOTYPE_STATUS")
  
    autosomalAssocResults <- autosomalAssocResults[,..columnSubset]
      names(autosomalAssocResults) <- c("CHR", "POS", "majorAllele", "minorAllele", "SNPID", "BETA", "SE", "OR", "LogOR", "Lower95OR", "Upper95OR", 
                                        "MAF", "MAC", "p.value", "N", "negLog10pvalue",  "R2", "ER2", "GENOTYPE_STATUS")
    # write a copy of unfilterd table
      fwrite(autosomalAssocResults, file = paste(dataOutputPrefix, "GWASresults_allSNPs_noFiltering.txt", sep="_"), append=FALSE, col.names=TRUE, showProgress = T, nThread = nThreads, compress = "gzip", sep = "\t")
      #write_tsv(autosomalAssocResults, paste(dataOutputPrefix, "GWASresults_allSNPs_noFiltering.txt", sep="_"), append=FALSE, col_names=TRUE)
    
    # filter by MAC and MAF based on user input to generate filtered common variants file (>macFilter, >mafFilter)
    commonClean <- autosomalAssocResults[which(as.numeric(autosomalAssocResults$MAC) > macFilter & 
                                                 as.numeric(autosomalAssocResults$MAF) > mafFilter),]
    fwrite(commonClean, file = paste(dataOutputPrefix, "GWASresults_commonSNPs_cleaned.txt", sep="_"), append=FALSE, col.names=TRUE, showProgress = T, nThread = nThreads, compress = "gzip", sep = "\t")
    #write_tsv(commonClean, paste(dataOutputPrefix, "GWASresults_commonSNPs_cleaned.txt", sep="_"), append=FALSE, col_names=TRUE)

    
    # filter by MAC and MAF based on user input to generate filtered rare variants file (>macFilter, <=mafFilter)
    rareClean <- autosomalAssocResults[which(as.numeric(autosomalAssocResults$MAC) > macFilter & 
                                               as.numeric(autosomalAssocResults$MAF) <= mafFilter),]
    
    fwrite(rareClean, file = paste(dataOutputPrefix, "GWASresults_rareSNPs_cleaned.txt", sep="_"), append=FALSE, col.names=TRUE, showProgress = T, nThread = nThreads, compress = "gzip", sep = "\t")
    #write_tsv(rareClean, paste(dataOutputPrefix, "GWASresults_rareSNPs_cleaned.txt", sep="_"),append=FALSE, col_names=TRUE)

  }
 

  # common mac filter and maf filter
  observed <- sort(commonClean$p.value)
  lobs <- -(log10(observed))
  expected <- c(1:length(observed)) 
  lexp <- -(log10(expected / (length(expected)+1)))
  png(filename=paste(dataOutputPrefix, "qqplot_commonSNPs_cleaned.png", sep="_"), width=800, height=800, bg="white", type="cairo")
  par(mar=c(5,6,4,1) + .1)
  plot(c(0,10), c(0,10), col="red", lwd=3, type="l", xlab="Expected (-logP)", ylab="Observed (-logP)", xlim=c(0,10), ylim=c(0,10), las=1, xaxs="i", yaxs="i", bty="l", cex.axis=2.0, cex.lab=2.0)
  points(lexp, lobs, pch=23, cex=.4, bg="black") 
  #inflation factor lambda
  chisq2 <- qchisq(1- commonClean$p.value,1,lower.tail = T)
  lambda <- median(chisq2,na.rm=T)/qchisq(0.5,1)#lambda1
  mtext(bquote(paste("QQ Plot for SNPs MAC >", .(macFilter), " and MAF > ", .(mafFilter), ":  ", lambda == .(lambda))), side=3, cex=2.0)
  dev.off()
  
  
  # rare mac filter and maf filter
  observed <- sort(rareClean$p.value)
  lobs <- -(log10(observed))
  expected <- c(1:length(observed)) 
  lexp <- -(log10(expected / (length(expected)+1)))
  png(filename=paste(dataOutputPrefix, "qqplot_rareSNPs_cleaned.png", sep="_"), width=800, height=800, bg="white", type="cairo")
  par(mar=c(5,6,4,1) + .1)
  plot(c(0,10), c(0,10), col="red", lwd=3, type="l", xlab="Expected (-logP)", ylab="Observed (-logP)", xlim=c(0,10), ylim=c(0,10), las=1, xaxs="i", yaxs="i", bty="l", cex.axis=2.0, cex.lab=2.0)
  points(lexp, lobs, pch=23, cex=.4, bg="black") 
  #inflation factor lambda
  chisq2 <- qchisq(1- rareClean$p.value,1,lower.tail = T)
  lambda <- median(chisq2,na.rm=T)/qchisq(0.5,1)#lambda1
  mtext(bquote(paste("QQ Plot for SNPs MAC >", .(macFilter), " and MAF <= ", .(mafFilter), ":  ", lambda == .(lambda))), side=3, cex=2.0)
  dev.off()
  
  #common and clean Manhattan plot
  upperYlim <- max(c(15, max(commonClean$negLog10pvalue)))
  png(filename=paste(dataOutputPrefix, "manhattan_commonSNPs_cleaned.png", sep="_"), width=800, height=600, bg="white", type="cairo")
  par(font.axis = 2)
  substringPath <- strsplit(dataOutputPrefix, split = "/")
  prefix <- substringPath[[1]][length(substringPath[[1]])]
  manhattan(commonClean,chr = "CHR", 
            bp = "POS", 
            p = "p.value",
            col = c("gray60", "gray10"), 
            chrlabs = NULL,
            highlight = NULL, 
            logp = TRUE,
            suggestiveline = -log10(1e-05), 
            genomewideline = -log10(5e-08),
            ylim=c(0,upperYlim),
            cex.lab=1.5,
            cex.main=2.0,
            ylab="-log10(pvalue)",
            xlab="",
            annotateTop = T,
            main=paste(prefix, ": MAC >", macFilter, ", MAF>", mafFilter, sep=" ")
  )
  dev.off()
  
  # rare and clean manhattan plot
  upperYlim <- max(c(15, max(rareClean$negLog10pvalue)))
  png(filename=paste(dataOutputPrefix, "manhattan_rareSNPs_cleaned.png", sep="_"), width=800, height=600, bg="white", type="cairo", )
  par(font.axis = 2)
  substringPath <- strsplit(dataOutputPrefix, split = "/")
  prefix <- substringPath[[1]][length(substringPath[[1]])]
  manhattan(rareClean,chr = "CHR", 
            bp = "POS", 
            p = "p.value",
            col = c("gray60", "gray10"), 
            chrlabs = NULL,
            highlight = NULL, 
            logp = TRUE,
            suggestiveline = -log10(1e-05), 
            genomewideline = -log10(5e-08),
            ylim=c(0,upperYlim),
            cex.lab=1.5,
            cex.main=2.0,
            ylab="-log10(pvalue)",
            xlab="",
            annotateTop=T,
            main=paste(prefix, ": MAC >", macFilter, ", MAF <=", mafFilter, sep=" ")
  )
  dev.off()
  
  images <- lapply(list(paste(dataOutputPrefix, "qqplot_commonSNPs_cleaned.png", sep="_"),
                        paste(dataOutputPrefix, "qqplot_rareSNPs_cleaned.png", sep="_"),
                        paste(dataOutputPrefix, "manhattan_commonSNPs_cleaned.png", sep="_"),
                        paste(dataOutputPrefix, "manhattan_rareSNPs_cleaned.png", sep="_")),
                   png::readPNG)
  
  imageGrid <- lapply(images, grid::rasterGrob)

  tmp<-do.call(gridExtra::grid.arrange, c(imageGrid, top=paste(pheno, "~", stringr::str_replace_all(covars, ",", " + "), "\n cases: ", as.character(autosomalAssocResults$N.Cases[1]), 
                                                               "  controls:", as.character(autosomalAssocResults$N.Controls[1]), sep = " ")))
  
  
  ggsave(file=paste(dataOutputPrefix, "_finalGWASresults.pdf", sep = ""), tmp, width = 10, height = 10, units = "in")
}
