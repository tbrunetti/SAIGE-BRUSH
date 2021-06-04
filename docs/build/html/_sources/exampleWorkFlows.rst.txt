Tutorials and Examples
=======================

Quick Start Command
^^^^^^^^^^^^^^^^^^^

In order to run the pipeline, open a shell or bash prompt (or batch script for a job-scheduler) and run one of the following based on your host system:

Once all the files are accessible and the config is ready, the following command will run the pipeline.
For those running this through a **job-scheduler such as SLURM, LSF, PBS, etc...** the log and error files will output to the scheduler keyworkds for log and error so please set those in your job submission.  Then you can put the following line in your batch script to run the pipeline:

.. code-block:: bash 

	$ ./saigeBrush myConfigFile.txt 


For those running this ** without a job-scheduler** the log and error files will output/print to your screen/standard out.  Therefore, please specify log and error files by running the pipeline as follows:

.. code-block:: bash 

	$ ./saigeBrush myConfigFile.txt 1> myLogName.log 2> myLogName.err


Tutorials
^^^^^^^^^^
.. toctree::
   :maxdepth: 2

   fullPipelineBinaryTutorial
   generateGRMonlyTutorial
   generateNullonlyTutorial
   generateAssociationOnlyTutorial
   fullPipelineReuseChunksTutorial


