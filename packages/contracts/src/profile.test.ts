import { describe, expect, it } from 'vitest';

import {
  localAuthFormSchema,
  organizationApplicationSchema,
  profileSettingsSchema,
} from './profile';

describe('профиль и формы входа', () => {
  it('разрешает вход без имени, но требует имя при регистрации', () => {
    const values = {
      email: 'anna@example.test',
      password: 'СложныйПароль2026',
    };
    expect(
      localAuthFormSchema.safeParse({ ...values, mode: 'login' }).success,
    ).toBe(true);
    expect(
      localAuthFormSchema.safeParse({ ...values, mode: 'register', name: '' })
        .success,
    ).toBe(false);
  });
  it('не пропускает неподтвержденного представителя', () => {
    const values = {
      name: 'Шахматный клуб',
      description: 'Встречи для всех',
      representative: true,
    };
    expect(
      organizationApplicationSchema.safeParse({
        ...values,
        representative: false,
      }).success,
    ).toBe(false);
    const result = organizationApplicationSchema.parse(values);
    expect(result.representative).toBe(true);
  });
  it('проверяет интересы и все настройки каналов', () => {
    expect(
      profileSettingsSchema.safeParse({
        about: '',
        city: '',
        level: '',
        interests: [''],
        preferences: {
          large_text: false,
          reduced_motion: false,
          notify_replies: true,
          notify_decisions: true,
          notify_reminders: true,
        },
      }).success,
    ).toBe(false);
  });
});
