#!/bin/bash


firstEncounter=0
for assocs in ${1}*_SNPassociationAnalysis.txt
do
	if [ $firstEncounter -eq 0 ]
	then
		head -n 1 $assocs > ${1}"_header_allChromosomeResultsMerged.txt"
		tail -n+2 $assocs >> ${1}"_tmp_allChromosomeResultsMerged.txt"
		echo "$assocs finished merging. Exit status is $?."
		wait
		((firstEncounter++))
	fi
	tail -n+2 $assocs >> ${1}"_tmp_allChromosomeResultsMerged.txt"
	sort ${1}"_tmp_allChromosomeResultsMerged.txt" | uniq | > ${1}"_tmp_allChromosomeResultsMerged_sorted.txt"
	cat ${1}"_header_allChromosomeResultsMerged.txt" ${1}"_tmp_allChromosomeResultsMerged_sorted.txt" > ${1}"_allChromosomeResultsMerged.txt"
	rm ${1}"_tmp_allChromosomeResultsMerged.txt"
	rm ${1}"_header_allChromosomeResultsMerged.txt"
	echo "[func(main) -- Concatentate] $assocs finished merging. Exit status is $?."
	wait
done

echo "[func(main) -- Concatenate] All files have been merged into single file. Exit status is $?."

