package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/app"
	"fyne.io/fyne/widget"
	"fyne.io/fyne/theme"
	"fmt"
)



func main() {

	frontEnd := app.New() // creates a new app with appropriate drivers
	window := frontEnd.NewWindow("Welcome to the CCPM SAIGE GWAS Pipeline!") // creates a new window pop up with the listed string as the window title
	window.Resize(fyne.Size{800, 600})

	frontEnd.Settings().SetTheme(theme.LightTheme()) //for container run "go get fyne.io/fyne/cmd/fyne_settings" in container build

	fullPipelineForm := fullPipeline(window)
	stepOneForm := nullModel(window)
	stepTwoForm := associationAnalysis(window)
	stepThreeForm := compileResults(window)

	// set up different forms for different types of analysis
	tabs := widget.NewTabContainer(
		widget.NewTabItem("Welcome!", widget.NewLabel("Please visit github site: github.com/tbrunetti for more info use and documentation.")),
		widget.NewTabItem("Full pipeline", widget.NewScrollContainer(fullPipelineForm)),
		widget.NewTabItem("STEP1: Null Model Only", stepOneForm),
		widget.NewTabItem("STEP2: Association Analysis Only", stepTwoForm),
		widget.NewTabItem("STEP3: Clean and Compile Results Only", stepThreeForm))

	tabs.SetTabLocation(widget.TabLocationTop)
	
	window.SetContent(tabs)

	window.ShowAndRun()

}

func NewDialog(window fyne.Window) {
	var (
		selectedfiles fyne.URIReadCloser
		fileerror     error
	)

	dialog.ShowFileOpen(func(file fyne.URIReadCloser, err error) {
		selectedfiles = file
		err = fileerror
	}, window)

	window.Show()
}


func fullPipeline(window fyne.Window) fyne.Widget {
	test := widget.NewButton("Pick input file", func() {
			NewDialog(window)})

	phenoName := widget.NewEntry()
	phenoName.SetPlaceHolder("Exact name of the column for GWAS trait to be assessed")

	traitType := widget.NewRadio([]string{"binary trait", "quantitative trait"}, func(value string) {
			fmt.Println("Radio set to", value)
	})

	locoUse := widget.NewCheck("yes", func(value bool) {
		fmt.Println("check set to ", value)
	})

	invNormUse := widget.NewCheck("no", func(value bool) {
		fmt.Println("invNormUse is set to ", value)
	})

	inputFile := widget.NewEntry()
	inputFile.SetPlaceHolder("Full path to input file, including file name.")

	plinkFile := widget.NewEntry()
	plinkFile.SetPlaceHolder("Full path to plink file prefix.")

	covarsList := widget.NewEntry()
	covarsList.SetPlaceHolder("Comma-separted list of covariates to use.  Ex: pc1,pc2,pc3,sex")

	imputationFilePrefix := widget.NewEntry()
	imputationFilePrefix.SetPlaceHolder("Full path to input file prefix, including file name.")

	genomeBuild := widget.NewSelect([]string{"hg19", "GRCh38"}, func(value string) {
		fmt.Println("Human genome build set to ", value)
	})

	chromStart := widget.NewEntry()
	chromStart.SetText("1") // default value

	chromEnd := widget.NewEntry()
	chromEnd.SetText("22") // default value

	vcfField := widget.NewRadio([]string{"dosage", "genotype"}, func(value string) {
			fmt.Println("vcfField set to", value)
	})

	dropMissing := widget.NewCheck("yes", func(value bool) {
		fmt.Println("dropMissing set to ", value)
	})

	minMaf := widget.NewEntry()
	minMaf.SetText("0.0")

	minMac := widget.NewEntry()
	minMac.SetText("0")

	sparseKin := widget.NewCheck("yes", func(value bool) {
		fmt.Println("sparse kin set to ", value)
	})


	outNamePrefix := widget.NewEntry()
	outNamePrefix.SetText("/home/myResults")

	mafSplit := widget.NewEntry()
	mafSplit.SetText("0.05")

	filterMinMAC := widget.NewEntry()
	filterMinMAC.SetText("10")

	hla := widget.NewCheck("yes", func(value bool) {
		fmt.Println("HLA is set to ", value)
	})



	fullPipelineform := &widget.Form{
		OnCancel: func() {
			fmt.Println("Cancelled")
			window.Close()
		},
		OnSubmit: func() { // optional, handle form submission
			fmt.Println("Form submitted")
			window.Close()
		},
	}

	fullPipelineform.Append("Pick file: ", test)
	fullPipelineform.Append("Input Tab-Delimited File: ", inputFile)
	fullPipelineform.Append("Input Plink File Prefix: ", plinkFile)
	fullPipelineform.Append("Human genome build: ", genomeBuild)
	fullPipelineform.Append("GWAS Trait Type: ", traitType)
	fullPipelineform.Append("trait name:", phenoName)
	fullPipelineform.Append("covarites to use: ", covarsList)
	fullPipelineform.Append("Use inverse normalization?\n (quantitative traits only)", invNormUse)
	fullPipelineform.Append("Imputation File Path and Prefix: ", imputationFilePrefix)
	fullPipelineform.Append("Use LOCO?", locoUse)
	fullPipelineform.Append("Full output path including prefix:", outNamePrefix)
	fullPipelineform.Append("Starting Chromosome: ", chromStart)
	fullPipelineform.Append("Ending Chromosome: ", chromEnd)
	fullPipelineform.Append("Minimum MAF for Association Calculation:", minMaf)
	fullPipelineform.Append("Minimum MAC for Association Calculation:", minMac)
	fullPipelineform.Append("Calculate association on: ", vcfField)
	fullPipelineform.Append("drop missing dosages?", dropMissing)
	fullPipelineform.Append("Use Sparse Kin? (recommended)", sparseKin)
	fullPipelineform.Append("MAF to split common vs rare variants:", mafSplit)
	fullPipelineform.Append("MAC filter", filterMinMAC)
	fullPipelineform.Append("Are results HLA specific?", hla)
	
	return fullPipelineform
}


func nullModel(window fyne.Window) fyne.Widget {
	phenoName := widget.NewEntry()
	phenoName.SetPlaceHolder("Exact name of the column for GWAS trait to be assessed")

	traitType := widget.NewRadio([]string{"binary trait", "quantitative trait"}, func(value string) {
			fmt.Println("Radio set to", value)
	})

	locoUse := widget.NewCheck("yes", func(value bool) {
		fmt.Println("check set to ", value)
	})

	invNormUse := widget.NewCheck("no", func(value bool) {
		fmt.Println("invNormUse is set to ", value)
	})

	inputFile := widget.NewEntry()
	inputFile.SetPlaceHolder("Full path to input file, including file name.")

	plinkFile := widget.NewEntry()
	plinkFile.SetPlaceHolder("Full path to plink file prefix.")

	covarsList := widget.NewEntry()
	covarsList.SetPlaceHolder("Comma-separted list of covariates to use.  Ex: pc1,pc2,pc3,sex")



	stepOneForm := &widget.Form{
		OnCancel: func() {
			fmt.Println("Cancelled")
			window.Close()
		},
		OnSubmit: func() { // optional, handle form submission
			fmt.Println("Form submitted")
			window.Close()
		},
	}

	stepOneForm.Append("Input Tab-Delimited File: ", inputFile)
	stepOneForm.Append("Input Plink File Prefix: ", plinkFile)
	stepOneForm.Append("GWAS Trait Type: ", traitType)
	stepOneForm.Append("trait name:", phenoName)
	stepOneForm.Append("covarites to use: ", covarsList)
	stepOneForm.Append("Use LOCO?", locoUse)
	stepOneForm.Append("Use inverse normalization?\n (quantitative traits only)", invNormUse)

	return stepOneForm

}

func associationAnalysis(window fyne.Window) fyne.Widget {
	imputationFilePrefix := widget.NewEntry()
	imputationFilePrefix.SetPlaceHolder("Full path to input file prefix, including file name.")

	genomeBuild := widget.NewSelect([]string{"hg19", "GRCh38"}, func(value string) {
		fmt.Println("Human genome build set to ", value)
	})

	chromStart := widget.NewEntry()
	chromStart.SetText("1") // default value

	chromEnd := widget.NewEntry()
	chromEnd.SetText("22") // default value

	vcfField := widget.NewRadio([]string{"dosage", "genotype"}, func(value string) {
			fmt.Println("vcfField set to", value)
	})

	locoUse := widget.NewCheck("yes", func(value bool) {
		fmt.Println("loco set to ", value)
	})

	dropMissing := widget.NewCheck("yes", func(value bool) {
		fmt.Println("dropMissing set to ", value)
	})

	minMaf := widget.NewEntry()
	minMaf.SetText("0.0")

	minMac := widget.NewEntry()
	minMac.SetText("0")

	rdaFile := widget.NewEntry()
	rdaFile.SetPlaceHolder("/path/to/step1/rda/file.rda")

	varianceRatioFile := widget.NewEntry()
	varianceRatioFile.SetPlaceHolder("/path/to/step1/output/myVarianceRatio.txt")

	sparseKin := widget.NewCheck("yes", func(value bool) {
		fmt.Println("sparse kin set to ", value)
	})


	outNamePrefix := widget.NewEntry()
	outNamePrefix.SetText("/home/myResults")

	stepTwoForm := &widget.Form{
		OnCancel: func() {
			fmt.Println("Cancelled")
			window.Close()
		},
		OnSubmit: func() { // optional, handle form submission
			fmt.Println("Form submitted:", minMaf.Text)
			window.Close()
		},
	}

	stepTwoForm.Append("Imputation File Path and Prefix: ", imputationFilePrefix)
	stepTwoForm.Append("Full path to null model file from step1\n(ends in .rda)", rdaFile)
	stepTwoForm.Append("Full path to variance ratio file from step1\n(ends in varianceRatio.txt", varianceRatioFile)
	stepTwoForm.Append("Full output path including prefix:", outNamePrefix)
	stepTwoForm.Append("Human Genome Build: ", genomeBuild)
	stepTwoForm.Append("Starting Chromosome: ", chromStart)
	stepTwoForm.Append("Ending Chromosome: ", chromEnd)
	stepTwoForm.Append("Minimum MAF for Association Calculation:", minMaf)
	stepTwoForm.Append("Minimum MAC for Association Calculation:", minMac)
	stepTwoForm.Append("Calculate association on: ", vcfField)
	stepTwoForm.Append("use LOCO?", locoUse)
	stepTwoForm.Append("drop missing dosages?", dropMissing)
	stepTwoForm.Append("Use Sparse Kin? (recommended)", sparseKin)


	return stepTwoForm
}

func compileResults(window fyne.Window) fyne.Widget {
	traitType := widget.NewRadio([]string{"binary trait", "quantitative trait"}, func(value string) {
			fmt.Println("Radio set to", value)
	})
	
	mafSplit := widget.NewEntry()
	mafSplit.SetText("0.05")

	filterMinMAC := widget.NewEntry()
	filterMinMAC.SetText("10")

	hla := widget.NewCheck("yes", func(value bool) {
		fmt.Println("HLA is set to ", value)
	})

	stepThreeForm := &widget.Form{
		OnCancel: func() {
			fmt.Println("Cancelled")
			window.Close()
		},
		OnSubmit: func() { // optional, handle form submission
			fmt.Println("Form submitted")
			window.Close()
		},
	}

	stepThreeForm.Append("trait type:", traitType)
	stepThreeForm.Append("MAF to split common vs rare variants:", mafSplit)
	stepThreeForm.Append("MAC filter", filterMinMAC)
	stepThreeForm.Append("Are results HLA specific?", hla)


	return stepThreeForm
}