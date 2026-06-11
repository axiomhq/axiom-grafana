let initPromise: Promise<void> | undefined;

export function ensureMplInit(): Promise<void> {
  return (initPromise ??= import('@axiomhq/mpl').then((mpl) => mpl.default()).then(() => {}));
}
