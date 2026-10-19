/** Первая буква имени для круглого аватара: «Клуб «Ладья»» → «К». */
export const initialOf = (name: string) =>
  name
    .replace(/^[^\p{L}\p{N}]+/u, '')
    .charAt(0)
    .toUpperCase() || '·';
