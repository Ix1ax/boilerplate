/**
 * Тип-уровневый аналог deepCamelKeys: превращает snake_case поля сгенерированных
 * типов (@starter/contracts) в camelCase — то, что реально приходит из API после
 * преобразования в shared/api/client.ts. Руками не пишем camelCase-копии типов.
 */
export type CamelCase<S extends string> =
  S extends `${infer Head}_${infer Char}${infer Tail}`
    ? `${Head}${Uppercase<Char>}${CamelCase<Tail>}`
    : S;

export type Camelize<T> = T extends readonly (infer U)[]
  ? Camelize<U>[]
  : T extends Date
    ? T
    : T extends object
      ? { [K in keyof T as K extends string ? CamelCase<K> : K]: Camelize<T[K]> }
      : T;
