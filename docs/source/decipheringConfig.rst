Deciphering the config file
============================

The config file is a text file that contains key-value mappings to all input parameters and pipeline logic. The file can be broken down into three main parts: the pipeline logic, environment setup, and the user data input options.

Pipeline Logic
^^^^^^^^^^^^^^^
There are five keywords that control the logic of the pipeline and all five accept only boolean arguments (true/false):: 
	
	GenerateGRM:true
	GenerateNull:true
	GenerateAssociations:true
	GenerateResults:true
	SkipChunking:false


Environment Setup
^^^^^^^^^^^^^^^^^
There are three keywords that control the environment setup of the pipeline. It is a requirement to use the container provided as an LSF object on github. This ensures all software is properly versioned and it reduces the complexity of the pipeline by using predefining software paths as well as reducing installation issues that arise with many softawre dependencies. ::

	BindPoint:/path/to/bind/container
	BindPointTemp:/path/to/tmp/
	Container:/path/to/saige-brush-v039.sif

:code:`BindPoint` and :code:`BindPointTemp` are directories to where you want the container to be mounted.  :code:`BindPointTemp` allows you to give the container a secondary binding point where all temp files and calculations will be performed, and once finished will move the final files to the :code:`BindPoint`.  The files generated in :code:`BindPointTemp` will be deleted after the run completes.  


.. note::
	If you do not want to use or do not have a temp directory, please set this parameter to be the same as :code:`BindPoint`. 



.. warning::
	**IMPORTANT PLEASE READ!** The container follows the same rules of inheritance as Singularity specifies.  This means the :code:`BindPoint` and :code:`BindPointTemp` become the highest point in your directory tree.  Thus, you can only access paths, directories, and files if you are able to traverse below the tree from these starting points but not above these entry points from your host system.  Therefore, be sure all the paths, directories, and files specified in the config file are contained within the scope of at least one of these two entry points. 




User Data Input
^^^^^^^^^^^^^^^^
The remainder of the keywords are parameters offered to the user and the user can specify paths and options for all or some of these keywords: ::

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

User Data Input: Parameters that are paths to files
----------------------------------------------------
.. seealso::

	:doc:`parameters <parameters>` under the section: Full List of User Input Data Parameters.  This will prodvide keyword descriptions and types.  For file and name formatting of keyword values see :doc:`fileFormats <fileFormats>`.










