export { ErrorBoundary } from './ErrorBoundary';
export * from '@starter/ui';

// Фирменные и анимированные элементы лендинга «Вместе» — специфичны для клиента,
// в общий дизайн-кит (@starter/ui) не выносим.
export {
  AnimatedTextLines,
  type AnimatedTextLinesLayer,
  type AnimatedTextLinesProps,
  type AnimatedTextLinesSegment,
  type AnimatedTextLinesSlot,
} from './AnimatedTextLines';
export { LogoIcon, STOLOTO_PATH } from './icons';
export { BrandMark } from './BrandMark';
export { RadarRings } from './RadarRings';
export { Eyebrow } from './Eyebrow';
export { Chip } from './Chip';
export { HeroBand } from './HeroBand';
