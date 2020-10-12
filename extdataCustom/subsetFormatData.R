library(xlsx)
library(stringr)
library(plyr)

pcs <- read.table("~/Downloads/ADMIRE_GWAS_SAIGE/Biobank.v1.3.eigenvectors.070620.reordered.txt", header =T)
pcs['Subject_id'] <- stringr::str_replace(pcs$FULL_BBID, "WG[0-9].*-DNA-[A-H][0-9].*-", "")

compassData <- readxl::read_excel("~/Downloads/ADMIRE_GWAS_SAIGE/ADMIRE Genomics_Redeliver.xlsx", col_types = c("text", "text"))
clientPhenotype <- readxl::read_excel("~/Downloads/ADMIRE_GWAS_SAIGE/SAIGE_input_file.xlsx", sheet = "SAIGE_translation", skip = 1)
clientCompassMerge <- merge(clientPhenotype, compassData, by = "Arb_PersonID", all.x = T)
finalSAIGEformat<-join(x=pcs, y=clientCompassMerge, by="Subject_id", type="left")

write.table(finalSAIGEformat, "ADMIRE_AD_SAIGE_GWAS_input.txt", sep = "\t", row.names = F, quote = F)
