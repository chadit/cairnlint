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

		// Phase 3b: net/http rules
		noDefaultHTTPClientAnalyzer(),

		// Phase 4: grep check replacements
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
	}
}
