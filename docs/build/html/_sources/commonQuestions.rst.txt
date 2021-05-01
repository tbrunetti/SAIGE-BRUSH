Frequently Asked Questions
===========================

Is SAIGE-BRUSH able to handle non-autosomal chromsomes (X, Y, mitochondrial)?
""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""
At the moment SAIGE-BRUSH is limited to autosomal chromosomes only.

Is SAIGE-BRUSH able to handle other organisms besides human data?
""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""

SAIGE-BRUSH was designed with human genetic data in mind.  However, if base SAIGE v.0.39.0 is able to 
handle other organisms, one can still use the containerized full implementaion of SAIGE v.0.39.0
within the contianer, but it not be able to handle the go implementation or config file.  It will require the
user to use base SAIGE as intended by SAIGE docs.  SAIGE-BRUSH will only save you the time of installing a working version
of SAIGE by using the Singularity container.

Troubleshooting
================

:code:`error: inv_sympd(): matrix is singular or not positive definite` 
""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""
This usually means there are covariates or data in your phenotype file that have no variance or are collinear with eachother.
You should double check the phenotype and covariates listed your your config file and make sure there is no collinearity
or lack of variance. Once this is resolved, re-run the pipeline again.


:code:`glm.fit: fitted probabilities numerically 0 or 1 occurred`
""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""""

Similar to above, this usually means there are covariates or data in your phenotype file that have no variance or are collinear with eachother.
You should double check the phenotype and covariates listed your your config file and make sure there is no collinearity
or lack of variance. Once this is resolved, re-run the pipeline again.