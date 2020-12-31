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


Indices and tables
==================

* :ref:`genindex`
* :ref:`modindex`
* :ref:`search`
