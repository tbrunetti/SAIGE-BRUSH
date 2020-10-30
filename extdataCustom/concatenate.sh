#!/bin/bash


firstEncounter=0
for assocs in ${1}*_SNPassociationAnalysis.txt
do
	if [ $firstEncounter -eq 0 ]
	then
		head -n 1 $assocs > ${1}"_allChromosomeResultsMerged.txt"
		tail -n+2 $assocs >> ${1}"_allChromosomeResultsMerged.txt"
		echo "$assocs finished merging. Exit status is $?."
		wait
		((firstEncounter++))
	fi
	tail -n+2 $assocs >> ${1}"_allChromosomeResultsMerged.txt"
	echo "$assocs finished merging. Exit status is $?."
	wait
done

echo "All files have been merged into single file. Exit status is $?."

