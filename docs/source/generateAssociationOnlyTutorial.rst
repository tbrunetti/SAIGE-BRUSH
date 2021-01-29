Tutorial: Generate Association Analysis Only
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
This example will show you how to generate the association analysis step only.  It will guide you through how to properly set the logic, remind you to set the environment, list all the additional files you need, and finally which user parameters need to be set.

Section: Logic and Overview
----------------------------
**Association Analysis only** means you only want to run the association anlysis.  It makes an assumption that you already have the null model file (.rda) pre-calculated and have a pre-calculate variance ratio file (.varianceRatio.txt) and want to re-use/use it in this step by setting the keywords :code:`NullModelFile` and :code:`VarianceRatioFile` located in the config file.  These two files are the result of running :code:`GenerateNull:true`.

Choosing to run just the association analysis step is analagous to setting the pipeline logic kewords to the following: :: 	

	GenerateGRM:false
	GenerateNull:false
	GenerateAssociations:true
	GenerateResults:false

When :code:`GenerateAssociations:true`, the :code:`SkipChunking` logic comes into play.

.. note::
	This step produces the raw associaions results concatenated into a file.  It does not clean up the data, perform the proper flips, or generate graphs/figures.  If you want the raw data in addition to the previously mentioned actions, be sure to also set :code:`GenerateResults:true`.


Section: Step-by-Step Tutorial
-------------------------------



Section: Generated Output
--------------------------

.. seealso::

	For a interpreting and searching the log files for potential pipeline errors, see :doc:`Parsing Through StdErr and StdOut <parsingStdErrOut>`.

