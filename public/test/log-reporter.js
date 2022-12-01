const { t } = require('i18next');
const { result } = require('lodash');

class CustomReporter {
  constructor(globalConfig, reporterOptions, reporterContext) {
    this._globalConfig = globalConfig;
    this._options = reporterOptions;
    this._context = reporterContext;
  }

  onRunComplete(testContexts, results) {
    console.log('Custom reporter output:');
    console.log('global config: ', this._globalConfig);
    console.log('options for this reporter from Jest config: ', this._options);
    console.log('reporter context passed from test scheduler: ', this._context);
    console.log('results: ', results);

    const stats = {
      suites: results.numTotalTestSuites,
      tests: results.numTotalTests,
      passes: results.numPassedTests,
      pending: results.numPendingTests,
      failures: results.numFailedTests,
    };

    console.log(`JestStats ${objToLogAttributes(stats)}`);
    results.testResults.forEach((t) => {
      // JestTestResult title="should preserve the placement" suite="when generating the legend for a panel" file= duration=1 currentRetry=1
      printResult(t);
    });
  }
}

function printResult(result, file = '') {
  if (result.status === 'pending') {
    return;
  }
  if (result.testResults) {
    result.testResults.forEach((r) => printResult(r, r.file || file));
  } else {
    const testInfo = {
      title: result.title,
      suite: Array.isArray(result.ancestorTitles) ? result.ancestorTitles.join(' > ') : '',
      file,
      duration: result.duration,
      currentRetry: result.invocations,
    };
    console.log(`JestTestResult ${objToLogAttributes(testInfo)}`);
  }
}
/**
 * Stringify object to be log friendly
 * @param {Object} obj
 * @returns {String}
 */
function objToLogAttributes(obj) {
  return Object.entries(obj)
    .map(([key, value]) => `${key}=${formatValue(value)}`)
    .join(' ');
}

/**
 * Escape double quotes
 * @param {String} str
 * @returns
 */
function escapeQuotes(str) {
  return String(str).replaceAll('"', '\\"');
}

/**
 * Wrap the value within double quote if needed
 * @param {*} value
 * @returns
 */
function formatValue(value) {
  const hasWhiteSpaces = /\s/g.test(value);

  return hasWhiteSpaces ? `"${escapeQuotes(value)}"` : value;
}

module.exports = CustomReporter;
