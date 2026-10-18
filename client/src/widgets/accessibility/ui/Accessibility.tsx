import { russian } from 'open-accessibility-widget/languages';
import {
  AccessibilityWidget,
  builtInLanguages,
} from 'open-accessibility-widget/react';

const LANGUAGES = [
  ...builtInLanguages.filter((language) => language.code !== 'ar'),
  russian,
];

const A11Y_THEME = {
  primary: 'var(--primary)',
  onPrimary: 'var(--primary-foreground)',
  background: 'var(--popover)',
  surface: 'var(--secondary)',
  text: 'var(--popover-foreground)',
  mutedText: 'var(--muted-foreground)',
  border: 'var(--border)',
  radius: 'var(--radius)',
  fontFamily: 'var(--font-sans)',
};

export const Accessibility = () => {
  return (
    <AccessibilityWidget
      position="left"
      buttonSize={36}
      language="ru"
      languages={LANGUAGES}
      theme={A11Y_THEME}
    />
  );
};
