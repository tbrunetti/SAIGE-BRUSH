Tutorial: Generate GRM Only
============================

This example will show you how to generate the GRM step only.  It will guide you through how to properly set the logic, remind you to set the environment, list all the additional files you need, and finally which user parameters need to be set.

Section: Logic and Overview
-----------------------------
**GRM only** means you want to run the GRM step.  This is analagous to setting the pipeline logic kewords to the following: 

*Note,* :code:`SkipChunking` *can be set to either* :code:`true` *or* :code:`false` *because it is only used if* :code:`GenerateAssociation` *is set to* :code:`true`. :: 	

	GenerateGRM:true
	GenerateNull:false
	GenerateAssociations:false
	GenerateResults:false
	SkipChunking:false  

If the pipeline is set to the above logic, the following workflow will be executed:

	.. image:: images/grmOnly_example.png
	   :width: 400
	   :align: center


Section: Step-by-Step Tutorial
-------------------------------

STEP 1: Set the logic
~~~~~~~~~~~~~~~~~~~~~

As stated about above, open your config file (.txt) and make sure the logic is set to the following: :: 

	GenerateGRM:true
	GenerateNull:false
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

For running the GRM only, you will need access to the following files:

#. **LD-pruned plink file**
	* used for when logic parameters :code:`GenerateGRM` is set to true and/or :code:`GenerateNull` is set to true.
	* fulfills parameter :code:`Plink`
	* see :ref:`Plink-File-Format` for formatting

.. seealso::

	For a interpreting and searching the log files for potential pipeline errors, see :doc:`Parsing Through StdErr and StdOut <parsingStdErrOut>`.


STEP 4: Set the path and values to all the required input parameters
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Now that you have all the required files, it is time to set the values and locations within your config file using the keywords expected.  Here are the required keywords and how to specify them:  

#. This :code:`RUNTYPE` parameter need to just be here for placeholder purposes, however it is required.  It has no impact on the pipeline, except as a header to check that it exists. :: 

	RUNTYPE:FULL

#. The next set of parameters are the keywords that relate to file inputs: 

	.. image:: images/grmOnly_fileparamters.png
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

	.. image:: images/grmOnly_output.png
		:width: 1000
		:align: center


.. warning::
	**IMPORTANT PLEASE READ!** Although the pipeline tries its best to not generate output as critical errors occur, this is not always the case.  It is particularly important to parse through the standard error output, as well as the log file produced in the :code:`other` directory of your output directory.  The log file can be quite large, therefore, it is recommended to use :code:`grep` to seach for keywords.  I would recommend the following: :code:`grep -i "err" other/*.log`, :code:`grep -i "warn" other/*.log`, and :code:`grep -i "exit" other/.*log`.  Also, please see the note below, for additional ways to parse the log file.


.. seealso::

	For a interpreting and searching the log files for potential pipeline errors, see :doc:`Parsing Through StdErr and StdOut <parsingStdErrOut>`.


Once it is confirmed that the error and log files ran successfully without major errors, the results and files are ready for viewing.  The directory of highest interest will be the :code:`grm_files` directory.

	.. image:: images/grmOnly_output_results.png
		:width: 1000
		:align: center

These files can be opened in R. The `.mtx` files are just sparse matrix files.  The :code:`*.sampleIDs.txt` file is important to keep and not mutate!  This is the order of the samples in the sparse matrix.  Therefore, do not shuffle or mutate this file and make sure to keep it together with the :code:`.mtx` files.


Section: Re-Use Next Steps
--------------------------
The output generated here can now be used as input file parameters in the config file for another run that requires the GRM input.  If you use these files for another run you can now set the logic parameter :code:`GenerateGRM:` to :code:`false` since you no longer need to calculate the GRM and are going to reuse the GRM that was just calculated by setting the following parameters:

	.. image:: images/grmOnly_output_results_nextSteps.png
		:width: 1000
		:align: center