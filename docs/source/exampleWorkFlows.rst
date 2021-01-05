Example Work Flows
===================

Quick Start Command
^^^^^^^^^^^^^^^^^^^

In order to run the pipeline, open a shell or bash prompt (or batch script for a job-scheduler) and type:

.. code-block:: bash 

	$ ./CCPM_GWAS_pipeline myConfigFile.txt 






Full Pipeline
^^^^^^^^^^^^^
**Full pipline** means you want to run every component of the pipeline from beginning to end in one go, without re-using any previously calculated data from the pipeline.  This is analagous to setting the pipeline logic kewords to the following: :: 	

	GenerateGRM:true
	GenerateNull:true
	GenerateAssociations:true
	GenerateResults:true
	SkipChunking:false



GRM only
^^^^^^^^^
**GRM only** means you want to run the GRM step.  This is analagous to setting the pipeline logic kewords to the following: 

*Note,* :code:`SkipChunking` *can be set to either* :code:`true` *or* :code:`false` *because it is only used if* :code:`GenerateAssociation` *is set to* :code:`true`. :: 	

	GenerateGRM:true
	GenerateNull:false
	GenerateAssociations:false
	GenerateResults:false
	SkipChunking:false


Null Model only
^^^^^^^^^^^^^^^
**Null Model only** means you want to run the Null Model only.  It makes an assumption that you already have the GRM pre-calculated and want to re-use it in this step by setting the keywords :code:`SparseGRM` and :code:`SampleIDFile` located in the config file.  These two files are the result of running :code:`GenerateGRM:true`.

Choosing to run just the null model generation step is analagous to setting the pipeline logic kewords to the following: :: 	

	GenerateGRM:false
	GenerateNull:true
	GenerateAssociations:false
	GenerateResults:false
	SkipChunking:false

:code:`SkipChunking` *can be set to either* :code:`true` *or* :code:`false` *because it is only used if* :code:`GenerateAssociation` *is set to* :code:`true`. 


Association Analyses Only
^^^^^^^^^^^^^^^^^^^^^^^^^^
**Association Analyses only** means you only want to run the association anlysis.  It makes an assumption that you already have the null model file (.rda) pre-calculated and have a pre-calculate variance ratio file (.varianceRatio.txt) and want to re-use/use it in this step by setting the keywords :code:`NullModelFile` and :code:`VarianceRatioFile` located in the config file.  These two files are the result of running :code:`GenerateNull:true`.

Choosing to run just the association analysis step is analagous to setting the pipeline logic kewords to the following: :: 	

	GenerateGRM:false
	GenerateNull:false
	GenerateAssociations:true
	GenerateResults:false

When :code:`GenerateAssociations:true`, the :code:`SkipChunking` logic comes into play.

.. note::
	This step produces the raw associaions results concatenated into a file.  It does not clean up the data, perform the proper flips, or generate graphs/figures.  If you want the raw data in addition to the previously mentioned actions, be sure to also set :code:`GenerateResults:true`.



Results and Graphs Only
^^^^^^^^^^^^^^^^^^^^^^^^
**Results and Graphs Only** cleans up raw data that was previously generated from an association analysis and generates cleaned data in addition to some figures/graphs.  You can use any association analysis here as long as it meets the file formatting specifications for supplying results in :code:`AssociationFile`.




Reuse Previously Indexed and Chunked Files
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^


Combinations of Logic
^^^^^^^^^^^^^^^^^^^^^^