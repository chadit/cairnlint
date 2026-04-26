package analyzers

import "golang.org/x/tools/go/analysis"

// Category groups related analyzers for display in `cairnlint linters`.
// The ordering of categories and analyzers here drives both registration
// order and human-facing listings, so there is one source of truth.
type Category struct {
	Name      string
	Analyzers []*analysis.Analyzer
}

// Categories returns every standard analyzer grouped by category.
func Categories() []Category {
	return []Category{
		{
			Name: "Scope-dependent (synctest exemption)",
			Analyzers: []*analysis.Analyzer{
				synctestSleepAnalyzer(),
				contextBackgroundAnalyzer(),
				contextTODOAnalyzer(),
				wrappedContextBackgroundAnalyzer(),
			},
		},
		{
			Name: "Loop-body and structural",
			Analyzers: []*analysis.Analyzer{
				deferInLoopAnalyzer(),
				queryInLoopAnalyzer(),
				stringConcatInLoopAnalyzer(),
				preferBLoopAnalyzer(),
				dbQueryWithBareBackgroundAnalyzer(),
				noElseAnalyzer(),
			},
		},
		{
			Name: "Expression-level",
			Analyzers: []*analysis.Analyzer{
				mapPreallocAnalyzer(),
				noUnderscoreTestNamesAnalyzer(),
				noRuntimeNumGoroutineAnalyzer(),
				noGenericErrorAnalyzer(),
				noErrStrContainsAnalyzer(),
				noPanicInLibAnalyzer(),
				noContextInStructAnalyzer(),
				preferErrorsAsTypeAnalyzer(),
				preferFmtAppendfAnalyzer(),
				typeAssertNoCheckAnalyzer(),
				noTestifySuitesAnalyzer(),
				preferVarZeroAnalyzer(),
				reflectNoKindCheckAnalyzer(),
				bufferPeekStoreAnalyzer(),
				typedNilErrorAnalyzer(),
				reflectInLoopAnalyzer(),
				benchReportAllocsAnalyzer(),
				benchResetTimerAnalyzer(),
				builderGrowAnalyzer(),
			},
		},
		{
			Name: "net/http",
			Analyzers: []*analysis.Analyzer{
				noDefaultHTTPClientAnalyzer(),
				httpClientTimeoutAnalyzer(),
			},
		},
		{
			Name: "Concurrency",
			Analyzers: []*analysis.Analyzer{
				wgAddBeforeGoAnalyzer(),
				goWGGoAnalyzer(),
				wgDoneInWGGoAnalyzer(),
				preferWGGoAnalyzer(),
				tickerLeakAnalyzer(),
				chanDirectionAnalyzer(),
				chanDirCloseAnalyzer(),
				stmtNoCloseAnalyzer(),
				poolResetBeforePutAnalyzer(),
			},
		},
		{
			Name: "Code quality",
			Analyzers: []*analysis.Analyzer{
				signalHandlingAnalyzer(),
				commentedOutCodeAnalyzer(),
				discardedContextAnalyzer(),
				sentinelErrorsAnalyzer(),
				sqlInjectionAnalyzer(),
				externalTestPkgAnalyzer(),
				noExportTestAnalyzer(),
				noForTestFuncAnalyzer(),
				noAAACommentsAnalyzer(),
				noInlineMocksAnalyzer(),
				unattributedTODOAnalyzer(),
				testCryptoInProdAnalyzer(),
			},
		},
		{
			Name: "Documentation style",
			Analyzers: []*analysis.Analyzer{
				emdashAnalyzer(),
				docParamBlockAnalyzer(),
				docTutorialVoiceAnalyzer(),
				testStructuredBlockAnalyzer(),
			},
		},
	}
}

// All returns every analyzer registered in cairnlint.
func All() []*analysis.Analyzer {
	cats := Categories()

	var total int
	for _, cat := range cats {
		total += len(cat.Analyzers)
	}

	out := make([]*analysis.Analyzer, 0, total)
	for _, cat := range cats {
		out = append(out, cat.Analyzers...)
	}

	return out
}
