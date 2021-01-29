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

:code:`SkipChunking` *can be set to either* :code:`true` *or* :code:`false` *because it is only used if* :code:`GenerateAssociation` *is set to* :code:`true`. 


Section: Step-by-Step Tutorial
-------------------------------



Section: Generated Output
--------------------------

.. seealso::

	For a interpreting and searching the log files for potential pipeline errors, see :doc:`Parsing Through StdErr and StdOut <parsingStdErrOut>`.

