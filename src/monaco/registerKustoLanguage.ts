const kustoLanguageId = 'kusto';

let monacoInstance: any;
let registeredMonacoInstance: any;
let pendingMonacoInstance: any;
let registrationPromise: Promise<void> | undefined;

function ensureKustoContribution() {
  if (!monacoInstance) {
    return undefined as any;
  }

  if (registeredMonacoInstance === monacoInstance || pendingMonacoInstance === monacoInstance) {
    return registrationPromise as any;
  }

  (window as any).monaco = monacoInstance;
  pendingMonacoInstance = monacoInstance;

  registrationPromise = Promise.all([
    import(/* webpackChunkName: "kusto-monaco" */ 'vs/language/kusto/kustoMode'),
    import(/* webpackChunkName: "kusto-monaco" */ '@kusto/monaco-kusto/release/esm/monaco.contribution'),
  ])
    .then(([, { setupMonacoKusto }]) => {
      const isKustoRegistered = monacoInstance.languages
        .getLanguages()
        .some((language: { id: string }) => language.id === kustoLanguageId);

      if (!isKustoRegistered) {
        setupMonacoKusto(monacoInstance);
      }

      registeredMonacoInstance = monacoInstance;
      pendingMonacoInstance = undefined;
    })
    .catch((error) => {
      pendingMonacoInstance = undefined;
      throw error;
    });

  return registrationPromise as any;
}
export function registerKustoLanguage(monaco?: any) {
  if (monaco) {
    monacoInstance = monaco;
  }

  return ensureKustoContribution();
}
