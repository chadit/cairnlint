package analyzers

import "golang.org/x/tools/go/analysis"

// All returns every analyzer registered in cairnlint.
func All() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		// Phase 1: scope-dependent rules (synctest exemption)
		synctestSleepAnalyzer(),
		contextBackgroundAnalyzer(),
		contextTODOAnalyzer(),
		wrappedContextBackgroundAnalyzer(),

		// Phase 2: loop-body and structural rules
		deferInLoopAnalyzer(),
		queryInLoopAnalyzer(),
		stringConcatInLoopAnalyzer(),
		preferBLoopAnalyzer(),
		dbQueryWithBareBackgroundAnalyzer(),
		noElseAnalyzer(),

		// Phase 3: expression-level rules
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

		// Phase 3b: net/http rules
		noDefaultHTTPClientAnalyzer(),
		httpClientTimeoutAnalyzer(),

		// Phase 3c: concurrency rules
		wgAddBeforeGoAnalyzer(),
		goWGGoAnalyzer(),
		wgDoneInWGGoAnalyzer(),
		tickerLeakAnalyzer(),
		chanDirectionAnalyzer(),
		chanDirCloseAnalyzer(),
		stmtNoCloseAnalyzer(),
		poolResetBeforePutAnalyzer(),

		// Phase 4: code quality
		signalHandlingAnalyzer(),

		// Phase 4b: grep check replacements
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
	}
}
