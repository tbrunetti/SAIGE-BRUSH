Deciphering the config file
============================

The config file is a text file that contains key-value mappings to all input parameters and pipeline logic. The file can be broken down into three main parts: the pipeline logic, enviroment setup, and the user data input options.

Pipeline Logic
^^^^^^^^^^^^^^^
There are five keywords that control the logic of the pipeline and all four accept only boolean arguments (true/false):: 
	
	GenerateGRM:true
	GenerateNull:true
	GenerateAssociations:true
	GenerateResults:true
	SkipChunking:false


Environment Setup
^^^^^^^^^^^^^^^^^
There are three keywords that control the environment setup of the pipeline. It is a requirement to use the container provided as an LSF object on github. This ensures all versioning of softaware if properly versioned and it reduces the complexity of the pipeline by using predefining software paths as well as reducing installation issues that arise with some many softawre dependencies::

	BindPoint:/path/to/bind/container
	BindPointTemp:/path/to/tmp/
	Container:/path/to/SAIGE_v0.39_CCPM_biobank_singularity_recipe_file_11162020.simg

User Data Input
^^^^^^^^^^^^^^^^
The remainder of the keywords are parameters offered to the user and the user can specify paths and options for all or some of these keywords::

	ChromosomeLengthFile:/path/to/chromosomeLengths.txt
	Build:hg38
	Chromosomes:1-22
	ImputeSuffix:_rsq70_merged_renamed.vcf.gz
	ImputeDir:/path/to/imputed/data/directory/
	OutDir:/path/to/output/final/results
	OutPrefix:myGWAS
	PhenoFile:/path/to/phenotype/covariate/file
	Plink:/path/to/LDpruned/plink/file/prefix
	Trait:binary
	Pheno:myPhenotype
	InvNorm:FALSE
	Covars:PC1,PC2,PC3,PC4,PC5,age,sex
	SampleID:SampleIDcol
	NThreads:
	SparseKin:True
	Markers:30
	Rel:0.0625
	Loco:TRUE
	CovTransform:True
	VcfField:DS
	MAF:0.05
	MAC:10
	IsDropMissingDosages:FALSE
	InfoFile:/path/to/info/file
	SaveChunks:false
	GrmMAF:0.01
	ChunkVariants:1000000
	SaveAsTar:false
	ImputationFileList:/path/to/list/of/chunked/chromosomes/file.txt
	SparseGRM:/path/to/grm/file.mtx
	SampleIDFile:/path/to/grm/sample/id/file.mtx.SampleID.txt
	NullModelFile:/path/to/null/model/file.rda
	VarianceRatioFile:/path/to/variance/ratio/file.txt
	AssociationFile:/path/to/concatenated/association/results/file.txt

