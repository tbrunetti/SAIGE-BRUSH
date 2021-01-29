Welcome to the CCPM Biobank GWAS Pipeline documentation!
========================================================

Installation
^^^^^^^^^^^^^
Good news!  There is no installation required to use this pipeline and it should be OS agnostic.  There are only 2 true system dependencies and most HPCs and shared resources already have these dependencies installed:

*  Singularity version >= 3.0
*  Golang version >= go1.13.5 (has been tested on versions go1.13.5 and go1.15.2 on a Linux OS).

If not, you can `install Singularity`_ and `install Golang`_ using the links on your local computer.  Both should be available across OS platforms for Windows, MacOS, and Linux.

.. _install Singularity: https://sylabs.io/docs/
.. _install Golang: https://golang.org/doc/install


Full Pipeline Overview
^^^^^^^^^^^^^^^^^^^^^^^

.. image:: images/fullPipeline.png
   :width: 800



Getting Started
^^^^^^^^^^^^^^^
.. toctree::
   :maxdepth: 2

   decipheringConfig
   parameters
   fileFormats
   output
   parsingStdErrOut



Quick Start and Examples
^^^^^^^^^^^^^^^^^^^^^^^^
.. toctree::
   :maxdepth: 2

   exampleWorkFlows


FAQs
^^^^


Acknowledgements
================

The most important acknowledgment is for the group of developers, investigaters, scientists, and researchers of SAIGE .  This pipline is honestly just an automated wrapper to most of their work.  They have an excellent github page here at https://github.com/weizhouUMICH/SAIGE -- The entire team is very responsive to github questions.  They are tirelessy working on newer and more complex versions of SAIGE, and for that I must thank them.  It is an amazing piece of software!  Please read their publication:

*Zhou, W., Nielsen, J. B., Fritsche, L. G., Dey, R., Gabrielsen, M. E., Wolford, B. N., LeFaive, J., VandeHaar, P., Gagliano, S. A., Gifford, A., Bastarache, L. A., Wei, W. Q., Denny, J. C., Lin, M., Hveem, K., Kang, H. M., Abecasis, G. R., Willer, C. J., & Lee, S. (2018). Efficiently controlling for case-control imbalance and sample relatedness in large-scale genetic association studies. Nature genetics, 50(9), 1335–1341. https://doi.org/10.1038/s41588-018-0184-y*



All of this work could not be done without the full support of the Colorado Center for Personalized Medicine (CCPM) under the guidance of their Biobank, and the Translation Informatics Services (TIS) sector, among several input from exerpienced scientists and professor within CCPM whose expertise is in GWAS.

	.. image:: images/tis_logo.png
		:width: 200
		:align: center


Indices and tables
==================

* :ref:`genindex`
* :ref:`modindex`
* :ref:`search`
