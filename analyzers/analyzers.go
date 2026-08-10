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
				synctestSleepWaitAnalyzer(),
				synctestRealServerAnalyzer(),
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
				testUnderscorePrefixAnalyzer(),
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
				preferColonEqualsAnalyzer(),
				preferCutLastAnalyzer(),
				urlCloneAnalyzer(),
			},
		},
		{
			Name: "net/http",
			Analyzers: []*analysis.Analyzer{
				noDefaultHTTPClientAnalyzer(),
				httpClientTimeoutAnalyzer(),
				redundantBodyDrainAnalyzer(),
			},
		},
		{
			Name: "Concurrency",
			Analyzers: []*analysis.Analyzer{
				wgAddBeforeGoAnalyzer(),
				goWGGoAnalyzer(),
				wgDoneInWGGoAnalyzer(),
				preferWGGoFanoutAnalyzer(),
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
				redundantErrNilCheckAnalyzer(),
				preferFatalErrGateAnalyzer(),
				noDotImportAnalyzer(),
				contextFirstParamAnalyzer(),
				testHelperMarkerAnalyzer(),
				tlsConfigRandAnalyzer(),
				goDebugRemovedAnalyzer(),
			},
		},
		{
			Name: "Naming",
			Analyzers: []*analysis.Analyzer{
				selfReceiverAnalyzer(),
				genericPackageNameAnalyzer(),
				noGetterPrefixAnalyzer(),
				mixedReceiverAnalyzer(),
				consistentReceiverNameAnalyzer(),
				constMixedCapsAnalyzer(),
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
		{
			Name:      "Modernizers (golang.org/x/tools, report-only)",
			Analyzers: modernizeAnalyzers(),
		},
	}
}

// All returns every analyzer registered in cairnlint, with suggested fixes
// stripped.
//
// The stripping happens here rather than at the call site because every
// consumer has a driver that can apply fixes: the multichecker in main has
// -fix, and golangci-lint has --fix for the module plugin in [plugin]. Doing
// it once on the way out of this package means a new consumer cannot forget.
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

	return WrapWithoutFixes(out)
}
