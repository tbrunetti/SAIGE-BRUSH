# SAIGE-BRUSH
THe SAIGE-Biobank Re-Usable SAIGE Helper software implmentation.  Implementation of SAIGE GWAS software with modifications, scalability, and user-friendly implementation.

## Dependencies  
These are the required dependencies on the host system for which you will be running SAIGE-BRUSH:  
* [Singularity](https://sylabs.io/docs/) version >= 3.0  
* [Golang](https://golang.org/doc/install) version >= go1.13.5 (officially test on go1.13.5 and go1.15.2)


## Getting Started

1.  Download the Singularity image:
	```
	singularity pull library://tbrunetti/default/saige-brush:v039 
	```

2.  Clone this repository:
	```
	git clone https://github.com/tbrunetti/SAIGE-BRUSH.git
	```

3.  Find the go executable and optionally you can add it to your path or move the exeutable to the desired run location.  It is called `saigeBrush`:
	```
	cd SAIGE-BRUSH/bin
	chmod u+x saigeBrush
	```

4.  The config file is located in the `container` directory and named `configSAIGE.txt`:  
	```
	cd SAIGE-BRUSH/container
	```

5.  After these steps you now have downloaded everything you need and you can follow the instructions in the [SAIGE-BRUSH Docs](https://saige-brush.readthedocs.io/en/latest/) to get started on your analysis!  

## Test Data  
Coming Soon!


## Full Documentation  

Please visit the [SAIGE-BRUSH Docs](https://saige-brush.readthedocs.io/en/latest/) for detailed information on getting started and running the framework.
