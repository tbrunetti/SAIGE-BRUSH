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
A small binary test data set is available in the git repo under `test_env/test_data`.  This set can be run on a local computer/laptop and meant to illustrate proof of principle and that the pipeline properly works. It typically runs the [full binary pipline - see here for example](https://saige-brush.readthedocs.io/en/latest/fullPipelineBinaryTutorial.html) from beginning to end in about 5-10 minutes.  This was testing on a Ubuntu Linux OS laptop with 12 cores/threads. Memory footprint is very low, so a standard laptop should not run into any memory issues on the test set.

1.  Complete all tasks (1-3) in the [Getting Started Section above](#getting-started)  

2.  Navigate to the test_data directory:  
	```
	cd ../test_env/test_data
	```
3.  Open the file `test_1k_configSAIGE.txt` and replace all values where it requires `/path/to/` with the full path location to the file listed for the paratmer.  No need to change anything else in the config file, just the paths.  

4.  Run the pipeline!  
	```
	cd ../../bin/
	./saigeBrush test_1k_configSAIGE.txt 1> myLog.log 2> myLog.err
	```

5.  It will create a full set of output for a small uninteresting random portion of chromosome 22.  For more information on expected output and how to run the test data, visit the [test data section](https://saige-brush.readthedocs.io/en/latest/) of our documentation.  



## Full Documentation  

Please visit the [SAIGE-BRUSH Docs](https://saige-brush.readthedocs.io/en/latest/) for detailed information on getting started and running the framework.
