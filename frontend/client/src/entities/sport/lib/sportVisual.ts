// Обложки встреч — «радар-карточки»: цвет фона, цвет значка и колец зависят от вида спорта.
export type SportCover = {
  /** Фон обложки. */
  bg: string;
  /** Цвет значка. */
  ink: string;
  /** Цвет колец радара. */
  ring: string;
  /** Цвет текста поверх обложки. */
  text: string;
};

export type SportIconKey =
  | 'chess'
  | 'gym'
  | 'football'
  | 'run'
  | 'medal'
  | 'petanque'
  | 'volleyball'
  | 'bike'
  | 'swim'
  | 'climb'
  | 'walk'
  | 'ski'
  | 'yoga'
  | 'tennis'
  | 'other';

const COVERS = {
  stone: {
    bg: '#E3E6DE',
    ink: '#121212',
    ring: 'rgba(18,18,18,.16)',
    text: '#121212',
  },
  gold: {
    bg: '#FAD553',
    ink: '#121212',
    ring: 'rgba(18,18,18,.2)',
    text: '#121212',
  },
  ink: {
    bg: '#121212',
    ink: '#FAD553',
    ring: 'rgba(250,213,83,.3)',
    text: '#EEF0EA',
  },
  blue: {
    bg: '#E8E8FE',
    ink: '#2340D8',
    ring: 'rgba(35,64,216,.22)',
    text: '#121212',
  },
  night: {
    bg: '#1A1D24',
    ink: '#FAD553',
    ring: 'rgba(250,213,83,.28)',
    text: '#EEF0EA',
  },
  cream: {
    bg: '#FFF3C4',
    ink: '#121212',
    ring: 'rgba(18,18,18,.16)',
    text: '#121212',
  },
} satisfies Record<string, SportCover>;

const PALETTE = Object.values(COVERS);

// Ключевые слова — по началу корня, чтобы ловить словоформы и свои направления.
const RULES: Array<[RegExp, SportIconKey, SportCover]> = [
  [/шахмат|шашк/i, 'chess', COVERS.stone],
  [/заряд|фитнес|зал|воркаут/i, 'gym', COVERS.gold],
  [/футбол|баскет|мини-футб/i, 'football', COVERS.ink],
  [/бег|пробеж|марафон/i, 'run', COVERS.blue],
  [/гто|норматив|легкоатл/i, 'medal', COVERS.night],
  [/петанк|боулинг|кёрлинг|керлинг/i, 'petanque', COVERS.cream],
  [/волейб/i, 'volleyball', COVERS.gold],
  [/вело/i, 'bike', COVERS.blue],
  [/плаван|бассейн/i, 'swim', COVERS.blue],
  [/скал|альпин/i, 'climb', COVERS.night],
  [/ходьб|прогулк/i, 'walk', COVERS.stone],
  [/лыж|коньк/i, 'ski', COVERS.blue],
  [/йог|стретч|пилат/i, 'yoga', COVERS.cream],
  [/теннис|бадминт|сквош/i, 'tennis', COVERS.gold],
];

const hash = (value: string) =>
  Array.from(value).reduce(
    (acc, char) => (acc * 31 + char.charCodeAt(0)) | 0,
    7,
  );

export const getSportVisual = (
  name: string,
): { icon: SportIconKey; cover: SportCover } => {
  const rule = RULES.find(([pattern]) => pattern.test(name));
  if (rule) return { icon: rule[1], cover: rule[2] };
  return {
    icon: 'other',
    cover: PALETTE[Math.abs(hash(name)) % PALETTE.length],
  };
};
