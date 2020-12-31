Minimum Requirements to Run Association Analysis
================================================

To calculate the association analyis, :code:`GenerateAssociations:true` needs to be specified in the config file.  If it is set to :code:`false`, the pipeline assumes one of two things:

#. *The association file results are not needed because one is provided in the config file from from a previous calculation*
#. *The association analysis files are not needed at all for the pipeline*