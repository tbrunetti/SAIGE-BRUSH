Tutorial: Full Pipeline with Binary Trait
==========================================

This example will walk you through how to run the full pipeline when the phenotype for association analysis if for a binary trait.  It will guide you through how to properly set the logic, remind you to set the environment, list all the additional files you need, and finally which user parameters need to be set.

Section: Logic and Overview
----------------------------
**Full pipline** means you want to run every component of the pipeline from beginning to end in one go, without re-using any previously calculated data from the pipeline.  This is analagous to setting the pipeline logic kewords to the following: :: 	

	GenerateGRM:true
	GenerateNull:true
	GenerateAssociations:true
	GenerateResults:true
	SkipChunking:false

If the pipeline is set to the above logic, the following workflow will be executed:

.. image:: images/fullPipeline_example.png
   :width: 400
   :align: center

Section: Step-by-Step Tutorial
-------------------------------

STEP 1: Set the logic
~~~~~~~~~~~~~~~~~~~~~

As stated about above, open your config file (.txt) and make sure the logic is set to the following: :: 

	GenerateGRM:true
	GenerateNull:true
	GenerateAssociations:true
	GenerateResults:true
	SkipChunking:false

STEP 2: Set the environment
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Open your config file (.txt) and make sure you set the path to where the bind point, temp bind point, and  container image are located.  I suggest you set the :code:`BindPoint` keyword to the same path as where the container is located to avoid any confusion.  If you have a tmp directory you want to use as scratch space, set that path as well.  If this doesn't exist or you choose not to use it, set the keyword :code:`BindPointTemp` to be the same as the path listed in the keyword :code:`BindPoint`. :: 

	BindPoint:/path/to/bind/container
	BindPointTemp:/path/to/tmp/
	Container:/path/to/saige-brush-v039.sif


STEP 3: Ensure you have all the files required
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

For running the full pipeline, including chunking the imputation files, you will need access to the following files:

#. **LD-pruned plink file**
	* used for when logic parameters :code:`GenerateGRM` is set to true and/or :code:`GenerateNull` is set to true.
	* fulfills parameter :code:`Plink`
	* see :ref:`Plink-File-Format` for formatting


#. **phenotype and covariates file**
	* used for when logic parameter :code:`GenerateNull` is set to true
	* fulfills parameter :code:`PhenoFile`
	* see :ref:`Phenotype-File-Format` for formatting


#. **chromosome lengths file**
	* used for when logic parameter, :code:`SkipChunking` is set to true. 
	* fulfills parameter :code:`ChromosomeLengthFile`
	* see :ref:`Chromosome-Length-File-Format` for formatting


#. **imputation files properly named and formatted or genotype files formatted in same way as imputation files**
	* used for when logic paramters :code:`SkipChunking` is set to true and/or :code:`GenerateAssociations` is set to true.
	* fulfills parameter :code:`ImputeSuffix`
	* see :ref:`Imputation-Name-Format` for formatting


#. **SNP information file**
	* use for when logic parameter :code:`GenerateResults` is set to true
	* fulfills parameter :code: `InfoFile`
	* see :ref:`Info-File-Format` for formatting


.. seealso::

	For a complete list of files and name formatting of keyword values listed in the config file see :doc:`Formatting the Required Files <fileFormats>`.


STEP 4: Set the path and values to all the required input parameters
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Now that you have all the required files, it is time to set the values and locations within your config file using the keywords expected.  Here are the required keywords and how to specify them:  

#. This :code:`RUNTYPE` parameter need to just be here for placeholder purposes, however it is required.  It has no impact on the pipeline, except as a header to check that it exists. :: 

	RUNTYPE:FULL

#. The next set of parameters are the keywords that relate to file inputs: 

	.. image:: images/fullPipeline_fileparamters.png
		:width: 700
		:align: center

#. Here are some required general keyword parameters that need to be set:

	.. image:: images/fullPipeline_generalParameter.png
		:width: 700
		:align: center

#. The following two sets of keyword parameters affect the GRM step, i.e. :code:`GenerateGRM:true` :
	
	.. image:: images/fullPipeline_grmParametres.png
		:width: 700
		:align: center

#. The following sets of keyword parameters affect the null model step, i.e. :code:`GenerateNull:true` :

	.. image:: images/fullPipeline_nullParameters.png
		:width: 700
		:align: center

#. The following sets of keyword parameters affect the association analysis step, i.e. :code:`GenerateAssociations:true` :

	.. image:: images/fullPipeline_AssociationParameters.png
		:width: 700
		:align: center


#. The following sets of keyword parameters affect the results step, i.e. :code:`GenerateResults:true` :

	.. image:: images/fullPipeline_resultsParameters.png
		:width: 700
		:align: center


#. These parameters I recommend to keep as is, unless you are familiar with the pipeline and have a reason to change them:

	.. image:: images/fullPipeline_otherParameters.png
		:width: 700
		:align: center


STEP 5: Running the pipeline
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
To run the pipeline make sure the files are all accessible to the container relative to the bindpoint.

Once all the files are accessible and the config is ready, the following command will run the pipeline:

.. code-block:: bash 

	$ ./saigeBrush myConfigFile.txt 


Section: Generated Output
--------------------------

The following graphic shows how all the data generated from running the logic of this pipeline will be organized and which files are present.  One thing to notice is the list of files generated in each directory based on whether the pipeline logic is set to :code:`true` or :code:`false`.  Many of these outputs and be re-used under certain circumstances to save time and bypass running certain steps of the pipeline in the next run.

	.. image:: images/fullPipeline_output.png
		:width: 1000
		:align: center


.. warning::
	**IMPORTANT PLEASE READ!** Although the pipeline tries its best to not generate output as critical errors occur, this is not always the case.  It is particularly important to parse through the standard error output, as well as the log file produced in the :code:`other` directory of your output directory.  The log file can be quite large, therefore, it is recommended to use :code:`grep` to seach for keywords.  I would recommend the following: :code:`grep -i "err" other/*.log`, :code:`grep -i "warn" other/*.log`, and :code:`grep -i "exit" other/.*log`.  Also, please see the note below, for additional ways to parse the log file.


.. seealso::

	For a interpreting and searching the log files for potential pipeline errors, see :doc:`Parsing Through StdErr and StdOut <parsingStdErrOut>`.


Once it is confirmed that the error and log files ran successfully without major errors, the results and files are ready for viewing.  The directory of highest interest will be the :code:`association_analysis_results` directory.

	.. image:: images/fullPipeline_output_results.png
		:width: 1000
		:align: center

When :code:`GenerateAssociations:true`, the pipeline generates raw association analysis data of all SNPs.  This set of data does have the allele flips in place, it is uncleaned and unfiltered, unannotated, lacking additional calculations and will not generate any visuals.  The file is the :code:`*allChromosomeResultsMerged.txt` files.


Now, when :code:`GenerateResults:true`, it takes that file, :code:`*allChromosomeResultsMerged.txt` and applies allele flips to ensure allele2 is always the minor allele, cleans the data using the :code:`MAC` filter, annotates the data with ER2, R2, and whether the SNP/Indel is imputed/genotyed/both.  This will also split your data in to common vs rare variants as defined by :code:`MAF` and generate qqplots and Manhattan plots for each.  The plots are put in a pdf report, :code:`*finalGWASresults.pdf`.  Each plot is also reported as individual pngs so they can easily be embedded into presentations and documents.  Here is an example of one of the pdf reports:

	.. image:: images/fullPipeline_pdf_example.png
		:width: 400
		:align: center

If you open any of the :code:`.txt.gz` files in located in the :code:`association_analysis_results` directory produced by :code:`GenerateResults:true`, the following headers are listed for all the SNPs/indels, in a tab-delimited file:

=============== =======================================================================
Header 			Definintion
=============== =======================================================================
CHR             chromosome name/ID
POS             position in chromosome, build is based on input imputation file build
majorAllele     major allele based on the allele frequency of your project
minorAllele     minor allele based on the allele frequency of your project
SNPID           snpID/name
BETA            beta value
SE              standard error of the beta
OR              odds ratio
LogOR           log(odds ratio)
Lower95OR       the lower 95% confidence interval of the odds ratio
Upper95OR	    the upper 95% confidence interval of the odds ratio
MAF             minor allele frequency
MAC             minor allele count
p.value         pvalue significance of association (note, for GWAS sig p<5e-08)
N               total samples used in this snp analysis
N.Cases         total number of case samples
N.Controls      total number of control samples
casesHomMinor   total number of cases that have homozygous minor alleles
casesHet        total number of cases that are heterozygous
controlHomMinor total number of controls that have homozygous minor alleles
controlHet      total number of controls that are heterozygous
negLog10pvalue  -log10(p.value)
R2              imputation R2 quality
ER2             empirical R2 quality  -- only for genotyped variants
GENTOYPE_STATUS whether a SNP is genotyped/imputed/both
=============== =======================================================================
