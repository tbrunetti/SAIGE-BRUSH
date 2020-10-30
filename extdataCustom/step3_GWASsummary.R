#!/usr/bin/env Rscript

options(stringsAsFactors=F)

## load R libraries
library(SAIGE)
library(optparse)
source("/opt/cleanAndGraph.R")

print(sessionInfo())

## set list of cmd line arguments
option_list <- list(
  make_option("--assocFile", type="character", default="",
              help="String Path - Full path directory to all association files output"),
  make_option("--infoFile", type="character", default="",
              help="String Path to SAIGE info field file from imputation"),
  make_option("--dataOutputPrefix", type="character", default="GWASresults",
              help="String - (no whitespace) of prefix name to output results"),
  make_option("--pheno", type="character", default="",
              help="String - (no whitespace) of phenotype to assess"),
  make_option("--covars", type="character", default="",
              help="String - (no whitespace) comma-separated list of covariates"),
  make_option("--macFilter", type="integer", default=10,
              help="Integer - Minimum allowable minor allele count to part of GWAS. By default, 10"),
  make_option("--mafFilter", type="numeric", default="0.05",
              help="Float (between 0.0 - 1.0) - Minimum minor allele frequency to splity data from common vs rare variant definition [default='0.05']"),
  make_option('--traitType', type="character", default="",
              help="options are either binary or quantitative. This refers to the phenotype being tested"),
  make_option("--nThreads", type="integer", default="1",
              help="Integer - total threads to use")


)

parser <- OptionParser(usage="%prog [options]", option_list=option_list)
args <- parse_args(parser, positional_arguments = 0)
opt <- args$options
print(opt)


cleanAndGraph(assocFile = opt$assocFile,
              infoFile = opt$infoFile,
	      dataOutputPrefix = opt$dataOutputPrefix,
	      pheno = opt$pheno,
	      covars = opt$covars,
              macFilter = opt$macFilter,
              mafFilter = opt$mafFilter,
              traitType = tolower(opt$traitType),
	      nThreads = opt$nThreads
              )


