/** Зажимает число в диапазон [min, max]. */
export const clamp = (value: number, min: number, max: number): number =>
  Math.min(Math.max(value, min), max);

export const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 Б';

  const units = ['Б', 'КиБ', 'МиБ', 'ГиБ'];
  const exponent = clamp(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    0,
    units.length - 1,
  );
  const value = bytes / 1024 ** exponent;

  return `${exponent === 0 ? value : value.toFixed(1)} ${units[exponent]}`;
};
