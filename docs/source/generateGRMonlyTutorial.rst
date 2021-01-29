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


Section: Step-by-Step Tutorial
-------------------------------



Section: Generated Output
--------------------------

.. seealso::

	For a interpreting and searching the log files for potential pipeline errors, see :doc:`Parsing Through StdErr and StdOut <parsingStdErrOut>`.

