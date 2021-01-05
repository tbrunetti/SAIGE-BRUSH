Minimum Requirements to Run Null Model
======================================

To calculate the null model, :code:`GenerateNull:true` needs to be specified in the config file.  If it is set to :code:`false`, the pipeline assumes one of two things:

#. *The null model files are not needed because one is provided in the config file from from a previous calculation*
#. *The null model files are not needed at all for the scope of the pipeline logic*
