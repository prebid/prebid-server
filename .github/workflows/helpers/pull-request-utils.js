const synchronizeEvent = "synchronize",
  openedEvent = "opened",
  completedStatus = "completed",
  resultSize = 100,
  adminPermission = "admin",
  writePermission = "write"

class diffHelper {
  constructor(input) {
    this.owner = input.context.repo.owner
    this.repo = input.context.repo.repo
    this.github = input.github
    this.pullRequestNumber = input.context.payload.pull_request.number
    this.pullRequestEvent = input.event
    this.testName = input.testName
    this.fileNameFilter = !input.fileNameFilter ? () => true : input.fileNameFilter
    this.fileLineFilter = !input.fileLineFilter ? () => true : input.fileLineFilter
  }

  /*
    Checks whether the test defined by this.testName has been executed on the given commit
    @param {string} commit - commit SHA to check for test execution
    @returns {boolean} - returns true if the test has been executed on the commit, otherwise false
  */
  async #isTestExecutedOnCommit(commit) {
    const response = await this.github.rest.checks.listForRef({
      owner: this.owner,
      repo: this.repo,
      ref: commit,
    })

    return response.data.check_runs.some(
      ({ status, name }) => status === completedStatus && name === this.testName
    )
  }

  /*
    Retrieves the line numbers of added or updated lines in the provided files
    @param {Array} files - array of files containing their filename and patch
    @returns {Object} - object mapping filenames to arrays of line numbers indicating the added or updated lines
  */
  async #getDiffForFiles(files = []) {
    let diff = {}
    for (const { filename, patch } of files) {
      if (this.fileNameFilter(filename)) {
        const lines = patch.split("\n")
        if (lines.length === 1) {
          continue
        }

        let lineNumber
        for (const line of lines) {
          // Check if line is diff header
          //  example:
          //    @@ -1,3 +1,3 @@
          //    1    var a
          //    2
          //    3   - //test
          //    3   +var b
          // Here @@ -1,3 +1,3 @@ is diff header
          if (line.match(/@@\s.*?@@/) != null) {
            lineNumber = parseInt(line.match(/\+(\d+)/)[0])
            continue
          }

          // "-" prefix indicates line was deleted. So do not consider deleted line
          if (line.startsWith("-")) {
            continue
          }

          // "+"" prefix indicates line was added or updated. Include line number in diff details
          if (line.startsWith("+") && this.fileLineFilter(line)) {
            diff[filename] = diff[filename] || []
            diff[filename].push(lineNumber)
          }
          lineNumber++
        }
      }
    }
    return diff
  }

  /*
    Retrieves a list of commits that have not been checked by the test defined by this.testName
    @returns {Array} - array of commit SHAs that have not been checked by the test
  */
  async #getNonScannedCommits() {
    const { data } = await this.github.rest.pulls.listCommits({
      owner: this.owner,
      repo: this.repo,
      pull_number: this.pullRequestNumber,
      per_page: resultSize,
    })
    let nonScannedCommits = []

    // API returns commits in ascending order. Loop in reverse to quickly retrieve unchecked commits
    for (let i = data.length - 1; i >= 0; i--) {
      const { sha, parents } = data[i]

      // Commit can be merged master commit. Such commit have multiple parents
      // Do not consider such commit for building file diff
      if (parents.length > 1) {
        continue
      }

      const isTestExecuted = await this.#isTestExecutedOnCommit(sha)
      if (isTestExecuted) {
        // Remaining commits have been tested in previous scans. Therefore, do not need to be considered again
        break
      } else {
        nonScannedCommits.push(sha)
      }
    }

    // Reverse to return commits in ascending order. This is needed to build diff for commits in chronological order
    return nonScannedCommits.reverse()
  }

  /*
    Filters the commit diff to include only the files that are part of the PR diff
    @param {Array} commitDiff - array of line numbers representing lines added or updated in the commit
    @param {Array} prDiff - array of line numbers representing lines added or updated in the pull request
    @returns {Array} - filtered commit diff, including only the files that are part of the PR diff
  */
  async #filterCommitDiff(commitDiff = [], prDiff = []) {
    return commitDiff.filter((file) => prDiff.includes(file))
  }

  /*
    Builds the diff for the pull request, including both the changes in the pull request and the changes in non-scanned commits
    @returns {string} - json string representation of the pull request diff and the diff for non-scanned commits
  */
  async buildDiff() {
    const { data } = await this.github.rest.pulls.listFiles({
      owner: this.owner,
      repo: this.repo,
      pull_number: this.pullRequestNumber,
      per_page: resultSize,
    })

    const pullRequestDiff = await this.#getDiffForFiles(data)

    const nonScannedCommitsDiff =
      Object.keys(pullRequestDiff).length != 0 && this.pullRequestEvent === synchronizeEvent // The "synchronize" event implies that new commit are pushed after the pull request was opened
        ? await this.getNonScannedCommitDiff(pullRequestDiff)
        : {}

    const prDiffFiles = Object.keys(pullRequestDiff)
    const pullRequest = {
      hasChanges: prDiffFiles.length > 0,
      files: prDiffFiles.join(" "),
      diff: pullRequestDiff,
    }
    const uncheckedCommits = { diff: nonScannedCommitsDiff }
    return JSON.stringify({ pullRequest, uncheckedCommits })
  }

  /*
    Retrieves the diff for non-scanned commits by comparing their changes with the pull request diff
    @param {Object} pullRequestDiff - The diff of files in the pull request
    @returns {Object} - The diff of files in the non-scanned commits that are part of the pull request diff
   */
  async getNonScannedCommitDiff(pullRequestDiff) {
    let nonScannedCommitsDiff = {}
    // Retrieves list of commits that have not been scanned by the PR check
    const nonScannedCommits = await this.#getNonScannedCommits()
    for (const commit of nonScannedCommits) {
      const { data } = await this.github.rest.repos.getCommit({
        owner: this.owner,
        repo: this.repo,
        ref: commit,
      })

      const commitDiff = await this.#getDiffForFiles(data.files)
      const files = Object.keys(commitDiff)
      for (const file of files) {
        // Consider scenario where the changes made to a file in the initial commit are completely undone by subsequent commits
        // In such cases, the modifications from the initial commit should not be taken into account
        // If the changes were entirely removed, there should be no entry for the file in the pullRequestStats
        const filePRDiff = pullRequestDiff[file]
        if (!filePRDiff) {
          continue
        }

        // Consider scenario where changes made in the commit were partially removed or modified by subsequent commits
        // In such cases, include only those commit changes that are part of the pullRequestStats object
        // This ensures that only the changes that are reflected in the pull request are considered
        const changes = await this.#filterCommitDiff(commitDiff[file], filePRDiff)

        if (changes.length !== 0) {
          // Check if nonScannedCommitsDiff[file] exists, if not assign an empty array to it
          nonScannedCommitsDiff[file] = nonScannedCommitsDiff[file] || []
          // Combine the existing nonScannedCommitsDiff[file] array with the commit changes
          // Remove any duplicate elements using the Set data structure
          nonScannedCommitsDiff[file] = [
            ...new Set([...nonScannedCommitsDiff[file], ...changes]),
          ]
        }
      }
    }
    return nonScannedCommitsDiff
  }

  /*
    Retrieves a list of directories from GitHub pull request files
    @param {Function} directoryExtractor - The function used to extract the directory name from the filename
    @returns {Array} An array of unique directory names
  */
  async getDirectories(directoryExtractor = () => "") {
    const { data } = await this.github.rest.pulls.listFiles({
      owner: this.owner,
      repo: this.repo,
      pull_number: this.pullRequestNumber,
      per_page: resultSize,
    })

    const directories = []
    for (const { filename, status } of data) {
      const directory = directoryExtractor(filename, status)
      if (directory != "" && !directories.includes(directory)) {
        directories.push(directory)
      }
    }
    return directories
  }
}

class semgrepHelper {
  constructor(input) {
    this.owner = input.context.repo.owner
    this.repo = input.context.repo.repo
    this.github = input.github

    this.pullRequestNumber = input.context.payload.pull_request.number
    this.pullRequestEvent = input.event

    this.pullRequestDiff = input.diff.pullRequest.diff
    this.newCommitsDiff = input.diff.uncheckedCommits.diff

    this.semgrepErrors = []
    this.semgrepWarnings = []
    input.semgrepResult.forEach((res) => {
      res.severity === "High" ? this.semgrepErrors.push(res) : this.semgrepWarnings.push(res)
    })

    this.headSha = input.headSha
  }

  /*
    Retrieves the matching line number from the provided diff for a given file and range of lines
    @param {Object} range - object containing the file, start line, and end line to find a match
    @param {Object} diff - object containing file changes and corresponding line numbers
    @returns {number|null} - line number that matches the range within the diff, or null if no match is found
  */
  async #getMatchingLineFromDiff({ file, start, end }, diff) {
    const fileDiff = diff[file]
    if (!fileDiff) {
      return null
    }
    if (fileDiff.includes(start)) {
      return start
    }
    if (fileDiff.includes(end)) {
      return end
    }
    return null
  }

  /*
    Splits the semgrep results into different categories based on the scan
    @param {Array} semgrepResults - array of results reported by semgrep
    @returns {Object} - object containing the categorized semgrep results i.e results reported in previous scans and new results found in the current scan
  */
  async #splitSemgrepResultsByScan(semgrepResults = []) {
    const result = {
      nonDiff: [], // Errors or warnings found in files updated in pull request, but not part of sections that were modified in the pull request
      previous: [], // Errors or warnings found in previous semgrep scans
      current: [], // Errors or warnings found in current semgrep scan
    }

    for (const se of semgrepResults) {
      const prDiffLine = await this.#getMatchingLineFromDiff(se, this.pullRequestDiff)
      if (!prDiffLine) {
        result.nonDiff.push({ ...se })
        continue
      }

      switch (this.pullRequestEvent) {
        case openedEvent:
          // "Opened" event implies that this is the first check
          // Therefore, the error should be appended to the result.current
          result.current.push({ ...se, line: prDiffLine })
        case synchronizeEvent:
          const commitDiffLine = await this.#getMatchingLineFromDiff(se, this.newCommitsDiff)
          // Check if error or warning is part of current commit diff
          // If not then error or warning was reported in previous scans
          commitDiffLine != null
            ? result.current.push({ ...se, line: commitDiffLine })
            : result.previous.push({
                ...se,
                line: prDiffLine,
              })
      }
    }
    return result
  }

  /*
    Adds review comments based on the semgrep results to the current pull request
    @returns {Object} - object containing the count of unaddressed comments from the previous scan and the count of new comments from the current scan
  */
  async addReviewComments() {
    let result = {
      previousScan: { unAddressedComments: 0 },
      currentScan: { newComments: 0 },
    }

    if (this.semgrepErrors.length == 0 && this.semgrepWarnings.length == 0) {
      return result
    }

    const errors = await this.#splitSemgrepResultsByScan(this.semgrepErrors)
    if (errors.previous.length == 0 && errors.current.length == 0) {
      console.log("Semgrep did not find any errors in the current pull request changes")
    } else {
      for (const { message, file, line } of errors.current) {
        await this.github.rest.pulls.createReviewComment({
          owner: this.owner,
          repo: this.repo,
          pull_number: this.pullRequestNumber,
          commit_id: this.headSha,
          body: message,
          path: file,
          line: line,
        })
      }
      result.currentScan.newComments = errors.current.length
      if (this.pullRequestEvent == synchronizeEvent) {
        result.previousScan.unAddressedComments = errors.previous.length
      }
    }

    const warnings = await this.#splitSemgrepResultsByScan(this.semgrepWarnings)
    for (const { message, file, line } of warnings.current) {
      await this.github.rest.pulls.createReviewComment({
        owner: this.owner,
        repo: this.repo,
        pull_number: this.pullRequestNumber,
        commit_id: this.headSha,
        body: "Consider this as a suggestion. " + message,
        path: file,
        line: line,
      })
    }
    return result
  }
}

class coverageHelper {
  constructor(input) {
    this.owner = input.context.repo.owner
    this.repo = input.context.repo.repo
    this.github = input.github
    this.pullRequestNumber = input.context.payload.pull_request.number
    this.headSha = input.headSha
    this.previewBaseURL = `https://htmlpreview.github.io/?https://github.com/${this.owner}/${this.repo}/coverage-preview/${input.remoteCoverageDir}`
    this.tmpCoverDir = input.tmpCoverageDir
  }

  /*
    Adds a code coverage summary along with heatmap links and coverage data on pull request as comment
    @param {Array} directories - directory for which coverage summary will be added
   */
  async AddCoverageSummary(directories = []) {
    const fs = require("fs")
    const path = require("path")
    const { promisify } = require("util")
    const readFileAsync = promisify(fs.readFile)

    let body = "## Code coverage summary \n"
    body += "Note: \n"
    body +=
      "- Prebid team doesn't anticipate tests covering code paths that might result in marshal and unmarshal errors \n"
    body += `- Coverage summary encompasses all commits leading up to the latest one, ${this.headSha} \n`

    for (const directory of directories) {
      let url = `${this.previewBaseURL}/${directory}.html`
      try {
        const textFilePath = path.join(this.tmpCoverDir, `${directory}.txt`)
        const data = await readFileAsync(textFilePath, "utf8")

        body += `#### ${directory} \n`
        body += `Refer [here](${url}) for heat map coverage report \n`
        body += "\`\`\` \n"
        body += data
        body += "\n \`\`\` \n"
      } catch (err) {
        console.error(err)
        return
      }
    }

    await this.github.rest.issues.createComment({
      owner: this.owner,
      repo: this.repo,
      issue_number: this.pullRequestNumber,
      body: body,
    })
  }
}

class userHelper {
  constructor(input) {
    this.owner = input.context.repo.owner
    this.repo = input.context.repo.repo
    this.github = input.github
    this.user = input.user
  }

  /*
    Checks if the user has write permissions for the repository
    @returns {boolean} - returns true if the user has write permissions, otherwise false
  */
  async hasWritePermissions() {
    const { data } = await this.github.rest.repos.getCollaboratorPermissionLevel({
      owner: this.owner,
      repo: this.repo,
      username: this.user,
    })
    return data.permission === writePermission || data.permission === adminPermission
  }
}

module.exports = {
  diffHelper: (input) => new diffHelper(input),
  semgrepHelper: (input) => new semgrepHelper(input),
  coverageHelper: (input) => new coverageHelper(input),
  userHelper: (input) => new userHelper(input),
}
