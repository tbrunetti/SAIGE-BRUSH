library(GENESIS) # version 3.6 required
library(Matrix)
library(gdsfmt)
# genesisObj is a Rdata object that can be loaded.  It is the output of running the GENESIS kinship/pc/grm.  
#The funciton below accesses the covMatList slot in order to get kinship values

# needs to be run on bigmem; tested on 100GB of defq and failed memory
genesisToSaige <- function(genesisObj){
  load(genesisObj)
  tempSparse <- makeSparseMatrix(x = covMatList$Kin)
  dsTMatrixConvert <- as(tempSparse, "TsparseMatrix")
  if (dsTMatrixConvert@uplo == "U"){
    print("Coverting to upper triangle...")
    dsTMatrixConvertTranspose <- t(dsTMatrixConvert)
    save(dsTMatrixConvert, file="test_GENESISToSaigeGSDfunc.Rdata")
    return(dsTMatrixConvertTranspose)
  }else if(dsTMatrixConvert@uplo == "L"){
    print("Matrix confirmed as lower triangle...")
    save(dsTMatrixConvert, file="test_GENESISToSaigeGSDfunc.Rdata")
    return(dsTMatrixConvert)
  }else
    cat("Issue in genesisToSaige:  uplo is not U or L, please check.  Exiting and return non-zero exit")
    quit(save="no", status = 42)
}





#args <- commandArgs(trailingOnly=TRUE)

#genesisToSaige(genesisObj=args[1])



###-------------------------------------- TESTING BELOW HERE --------------------------------------###
#covmatlist <- openfn.gds("/gpfs/share/BiobankPrototype/CCPM_biobank_freeze1_final_08232019/BC_clean/tmp_pcrelate.gds")   
#kin <- index.gdsn(covmatlist, "kinship")
#kinMat <- read.gdsn(kin)


genesisToSaigeGDS <- function(genesisObj){
  covmatlist <- openfn.gds(genesisObj)   
  kin <- index.gdsn(covmatlist, "kinship")
  tempSparse <- makeSparseMatrix(x = read.gdsn(kin))
  dsTMatrixConvert <- as(tempSparse, "TsparseMatrix")
  if (dsTMatrixConvert@uplo == "U"){
    print("Coverting to upper triangle...")
    dsTMatrixConvertTranspose <- t(dsTMatrixConvert)
    save(dsTMatrixConvert, file="test_GENESISToSaigeGSDfunc.Rdata")
    return(dsTMatrixConvertTranspose)
  }else if(dsTMatrixConvert@uplo == "L"){
    print("Matrix confirmed as lower triangle...")
    save(dsTMatrixConvert, file="test_GENESISToSaigeGSDfunc.Rdata")
    return(dsTMatrixConvert)
  }else
    cat("Issue in genesisToSaige:  uplo is not U or L, please check.  Exiting and return non-zero exit")
    quit(save="no", status = 42)
}

args <- commandArgs(trailingOnly=TRUE)

genesisToSaigeGDS(genesisObj=args[1])
