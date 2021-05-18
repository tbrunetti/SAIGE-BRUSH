Tutorial: Generate Null Model only
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

This example will show you how to generate the null model step only.  It will guide you through how to properly set the logic, remind you to set the environment, list all the additional files you need, and finally which user parameters need to be set.

Section: Logic and Overview
-----------------------------
**Null Model only** means you want to run the Null Model only.  It makes an assumption that you already have the GRM pre-calculated and want to re-use it in this step by setting the keywords :code:`SparseGRM` and :code:`SampleIDFile` located in the config file.  These two files are the result of running :code:`GenerateGRM:true`.

Choosing to run just the null model generation step is analagous to setting the pipeline logic kewords to the following: :: 	

	GenerateGRM:false
	GenerateNull:true
	GenerateAssociations:false
	GenerateResults:false
	SkipChunking:false

:code:`SkipChunking` *can be set to either* :code:`true` *or* :code:`false` *because it is only used if* :code:`GenerateAssociation` *is set to* :code:`true`.   If the pipeline is set to the above logic, the following workflow will be executed:

	.. image:: images/nullOnly_example.png
		:width: 400
		:align: center


Section: Step-by-Step Tutorial
-------------------------------

STEP 1: Set the logic
~~~~~~~~~~~~~~~~~~~~~

As stated about above, open your config file (.txt) and make sure the logic is set to the following: :: 

	GenerateGRM:false
	GenerateNull:true
	GenerateAssociations:false
	GenerateResults:false
	SkipChunking:false  

STEP 2: Set the environment
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Open your config file (.txt) and make sure you set the path to where the bind point, temp bind point, and  container image are located.  I suggest you set the :code:`BindPoint` keyword to the same path as where the container is located to avoid any confusion.  If you have a tmp directory you want to use as scratch space, set that path as well.  If this doesn't exist or you choose not to use it, set the keyword :code:`BindPointTemp` to be the same as the path listed in the keyword :code:`BindPoint`. :: 

	BindPoint:/path/to/bind/container
	BindPointTemp:/path/to/tmp/
	Container:/path/to/saige-brush-v039.sif  


STEP 3: Ensure you have all the files required
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	
For running the null model only, you will need access to the following files:
	
	#. **LD-pruned plink file**
		* used for when logic parameters :code:`GenerateGRM` is set to true and/or :code:`GenerateNull` is set to true.
		* fulfills parameter :code:`Plink`
		* see :ref:`Plink-File-Format` for formatting
  
	#. **phenotype and covariates file**
		* used for when logic parameter :code:`GenerateNull` is set to true
		* fulfills parameter :code:`PhenoFile`
		* see :ref:`Phenotype-File-Format` for formatting
  
	#. **pre-calculated GRM with corresponding sample order file**
		* used for when logic parameter :code:`GenerateNull` is set to true
		* fulfills parameters :code:`SparseGRM` and :code:`SampleIDFile`
		* see :ref:`SparseGRM-File-Format` and :ref:`SampleID-File-Format` for formatting


	.. note::
		**Missing the pre-calculated GRM files?**  No problem, if you set the logic to :code:`GenerateGRM:true`, one will be calculated for you! Just make sure you also set the GRM parameters you want.  For more information on what parameters you need to fill out, see :doc:`Minimum requirements for Generating a GRM <grmParameters>` or look at the :doc:`GRM only tutorial <generateGRMonlyTutorial>`.

	
.. seealso::

	For a complete list of files and name formatting of keyword values listed in the config file see :doc:`Formatting the Required Files <fileFormats>`.  


STEP 4: Set the path and values to all the required input parameters
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Now that you have all the required files, it is time to set the values and locations within your config file using the keywords expected.  Here are the required keywords and how to specify them:  
	
#. This :code:`RUNTYPE` parameter need to just be here for placeholder purposes, however it is required.  It has no impact on the pipeline, except as a header to check that it exists. :: 
	
	RUNTYPE:FULL
	
#. The next set of parameters are the keywords that relate to file inputs: 

	.. image:: images/nullOnly_fileparamters.png
		:width: 700
		:align: center

#. Here are some required general keyword parameters that need to be set:

	.. image:: images/fullPipeline_generalParameter.png
		:width: 700
		:align: center

#. The following set of keyword parameters affect the Null Model step, i.e. :code:`GenerateNull:true` :
	
	.. image:: images/fullPipeline_nullParameters.png
		:width: 700
		:align: center
	
#. These parameters I recommend to keep as is, unless you are familiar with the pipeline and have a reason to change them:

	.. image:: images/fullPipeline_otherParameters.png
		:width: 700
		:align: center


STEP 5: Running the pipeline
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
To run the pipeline make sure the files are all accessible to the container relative to the bindpoint.
		

Once all the files are accessible and the config is ready, the following command will run the pipeline.
For those running this through a **job-scheduler such as SLURM, LSF, PBS, etc...** the log and error files will output to the scheduler keyworkds for log and error so please set those in your job submission.  Then you can put the following line in your batch script to run the pipeline:

.. code-block:: bash 

	$ ./saigeBrush myConfigFile.txt 


For those running this ** without a job-scheduler** the log and error files will output/print to your screen/standard out.  Therefore, please specify log and error files by running the pipeline as follows:

.. code-block:: bash 

	$ ./saigeBrush myConfigFile.txt 1> myLogName.log 2> myLogName.err



Section: Generated Output
--------------------------

The following graphic shows how all the data generated from running the logic of this pipeline will be organized and which files are present.  One thing to notice is the list of files generated in each directory based on whether the pipeline logic is set to :code:`true` or :code:`false`.  Many of these outputs and be re-used under certain circumstances to save time and bypass running certain steps of the pipeline in the next run.

	.. image:: images/nullOnly_output.png
		:width: 1000
		:align: center


.. warning::
	**IMPORTANT PLEASE READ!** Although the pipeline tries its best to not generate output as critical errors occur, this is not always the case.  It is particularly important to parse through the standard error output, as well as the log file produced in the :code:`other` directory of your output directory.  The log file can be quite large, therefore, it is recommended to use :code:`grep` to seach for keywords.  I would recommend the following: :code:`grep -i "err" other/*.log`, :code:`grep -i "warn" other/*.log`, and :code:`grep -i "exit" other/.*log`.  Also, please see the note below, for additional ways to parse the log file.


.. seealso::

	For a interpreting and searching the log files for potential pipeline errors, see :doc:`Parsing Through StdErr and StdOut <parsingStdErrOut>`.


Once it is confirmed that the error and log files ran successfully without major errors, the results and files are ready for viewing.  The directory of highest interest will be the :code:`null_model_files` directory.

	.. image:: images/nullOnly_output_results.png
		:width: 1000
		:align: center


The null model file :code:`*.rda` is a binary file that can be opened in R.  It contains all the information generated to fit a null model.  This needs to be recalculated for each phenotype or if the covariates change. 
The :code:`*.varianceRatio.txt` file is a human-readable text file that contains a single value based on the x number of random markers listed in the config file as :code:`Markers` chosen to calculate this variance ratio.
Similar to the :code:`*.rda`  file, this needs to be calculated everytime the phenotype changes and the covariates used changes.


Section: Re-Use Next Steps
--------------------------
The output generated here can now be used as input file parameters in the config file for another run that requires the null model files as input.  If you use these files for another run you can now set the logic parameter :code:`GenerateNull:` to :code:`false` since you no longer need to calculate the null model and are going to reuse the null model that was just calculated by setting the following parameters:

	.. image:: images/nullOnly_output_results_nextSteps.png
		:width: 1000
		:align: center