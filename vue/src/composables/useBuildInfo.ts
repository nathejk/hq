export interface BuildInfo {
  gitCommit: string
  gitBranch: string
  buildNumber: string
}

function getMetaContent(name: string, fallback: string): string {
  const value = document.querySelector(`meta[name="${name}"]`)?.getAttribute('content') || ''
  if (!value || value.startsWith('__BUILD_')) return fallback
  return value
}

export function useBuildInfo(): BuildInfo {
  return {
    gitCommit: getMetaContent('git-commit', 'dev'),
    gitBranch: getMetaContent('git-branch', 'dev'),
    buildNumber: getMetaContent('build-number', '000')
  }
}
