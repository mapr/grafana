const { t } = require('i18next');
const { result } = require('lodash');

class CustomReporter {
  constructor(globalConfig, reporterOptions, reporterContext) {
    this._globalConfig = globalConfig;
    this._options = reporterOptions;
    this._context = reporterContext;
  }

  onRunComplete(testContexts, results) {
    if (!this._options.enable) {
      return;
    }

    console.log('Results:', results);
    this.logStats(results);
    this.logResults(results);
  }

  logResults(results) {
    results.testResults.forEach((t) => {
      // JestTestResult title="should preserve the placement" suite="when generating the legend for a panel" file= duration=1 currentRetry=1
      printTestResult(t, result.testFilePath);
    });
  }

  logStats(results) {
    const stats = {
      suites: results.numTotalTestSuites,
      tests: results.numTotalTests,
      passes: results.numPassedTests,
      pending: results.numPendingTests,
      failures: results.numFailedTests,
    };

    console.log(`JestStats ${objToLogAttributes(stats)}`);
  }
}

function printTestResult(result, file = '') {
  if (result.status === 'pending') {
    return;
  }
  if (result.testResults) {
    result.testResults.forEach((r) => printTestResult(r, file || result.testFilePath));
  } else {
    const testInfo = {
      title: result.title,
      suite: Array.isArray(result.ancestorTitles) ? result.ancestorTitles.join(' > ') : '',
      file,
      duration: result.duration,
      currentRetry: result.invocations,
    };
    console.log(`JestTestResult ${objToLogAttributes(testInfo)}`);
    if (result.status === 'failure') {
    }
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
