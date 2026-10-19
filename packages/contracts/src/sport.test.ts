import { describe, expect, it } from 'vitest';

import {
  type SportEvent,
  changePasswordSchema,
  companyRequestSchema,
  eventInputSchema,
  eventEditSchema,
  eventPatchSchema,
  joinAction,
  joinSchema,
  loginSchema,
  profilePatchSchema,
  registerSchema,
  reportSchema,
} from './sport';

const future = new Date(Date.now() + 48 * 3600_000).toISOString();
const valid = {
  title: 'Пробежка',
  sport_name: '  Ультимат   фрисби ',
  starts_at: future,
  location: 'Набережная',
  join_mode: 'open',
};

describe('eventInputSchema', () => {
  it('принимает минимальную форму и нормализует своё направление', () => {
    const v = eventInputSchema.parse(valid);
    expect(v.sport_name).toBe('Ультимат фрисби');
    expect(v.level).toBe('any');
    expect(v.as_organization).toBe(false);
  });

  it('требует направление, будущую дату и минимум не больше мест', () => {
    const issues = (input: object) =>
      eventInputSchema
        .safeParse({ ...valid, ...input })
        .error?.issues.map((i) => i.path.join('.'));
    expect(issues({ sport_name: undefined })).toContain('sport');
    expect(
      issues({ starts_at: new Date(Date.now() - 60_000).toISOString() }),
    ).toContain('starts_at');
    expect(issues({ capacity: 4, min_participants: 6 })).toContain(
      'min_participants',
    );
    expect(issues({ lat: 56 })).toContain('location');
    expect(issues({ title: 'Йо' })).toContain('title');
  });
});

describe('проверка временных границ встречи', () => {
  it('отклоняет прошедший срок сбора и срок после начала', () => {
    for (const deadline of [
      new Date(Date.now() - 60_000),
      new Date(new Date(future).getTime() + 60_000),
    ]) {
      const result = eventInputSchema.safeParse({
        ...valid,
        min_participants: 2,
        quorum_deadline: deadline.toISOString(),
      });
      expect(
        result.error?.issues.map((issue) => issue.path.join('.')),
      ).toContain('quorum_deadline');
    }
  });
  it('не даёт очистить начало в форме правки и переставить окончание раньше него', () => {
    expect(eventEditSchema.safeParse({ starts_at: undefined }).success).toBe(
      false,
    );
    const result = eventEditSchema.safeParse({
      starts_at: future,
      ends_at: new Date(new Date(future).getTime() - 60_000).toISOString(),
    });
    expect(result.error?.issues.map((issue) => issue.path.join('.'))).toContain(
      'ends_at',
    );
  });
});

describe('eventPatchSchema', () => {
  it('не подставляет значения по умолчанию в непереданные поля', () => {
    expect(eventPatchSchema.parse({ title: 'Новое имя' })).toEqual({
      title: 'Новое имя',
    });
  });
});

describe('другие формы', () => {
  it('запись по умолчанию — не новичок', () => {
    expect(joinSchema.parse({})).toEqual({ message: '', first_time: false });
  });

  it('радиус «Ищу компанию» — только с точкой', () => {
    const r = companyRequestSchema.safeParse({
      sport_name: 'Бег',
      radius_km: 5,
    });
    expect(r.success).toBe(false);
    expect(
      companyRequestSchema.parse({
        sport_name: 'Бег',
        lat: 56,
        lng: 92,
        radius_km: 5,
      }).day_parts,
    ).toEqual([]);
  });

  it('пустой уровень профиля — «не указывать»', () => {
    expect(profilePatchSchema.parse({ level: '' })).toEqual({ level: '' });
    expect(profilePatchSchema.safeParse({ level: 'master' }).success).toBe(
      false,
    );
  });
});

describe('joinAction', () => {
  const base = {
    status: 'published',
    join_mode: 'open',
    is_organizer: false,
    my_participation: null,
    spots_left: null,
    starts_at: future,
    ends_at: null,
  } as unknown as SportEvent;
  const with_ = (patch: object) => ({ ...base, ...patch }) as SportEvent;

  it('выбирает действие по состоянию события и записи', () => {
    expect(joinAction(base)).toBe('join');
    expect(joinAction(with_({ join_mode: 'request' }))).toBe('apply');
    expect(joinAction(with_({ spots_left: 0 }))).toBe('waitlist');
    expect(joinAction(with_({ is_organizer: true }))).toBe('manage');
    expect(joinAction(with_({ status: 'unpublished' }))).toBe('closed');
    expect(
      joinAction(
        with_({ starts_at: new Date(Date.now() - 3600_000).toISOString() }),
      ),
    ).toBe('closed');
    expect(joinAction(with_({ my_participation: { status: 'pending' } }))).toBe(
      'leave',
    );
    // Отказ окончателен на сервере: кнопки «записаться снова» быть не должно.
    expect(
      joinAction(with_({ my_participation: { status: 'rejected' } })),
    ).toBe('refused');
    expect(
      joinAction(with_({ my_participation: { status: 'cancelled' } })),
    ).toBe('rejoin');
  });
});

describe('вход и регистрация', () => {
  it('нормализует почту и проверяет пароль как identity', () => {
    const v = registerSchema.parse({
      email: '  Anna@Example.RU ',
      password: 'Зарядка2026',
      name: ' Анна ',
    });
    expect(v.email).toBe('anna@example.ru');
    expect(v.name).toBe('Анна');
    for (const bad of ['короткий1', 'onlyletters', '1234567890']) {
      const r = registerSchema.safeParse({
        email: 'a@b.ru',
        password: bad,
        name: 'Анна',
      });
      if (bad === 'короткий1') expect(r.success).toBe(true);
      else expect(r.error?.issues[0]?.path).toEqual(['password']);
    }
    expect(
      loginSchema.safeParse({ email: 'nope', password: 'x' }).success,
    ).toBe(false);
    expect(
      changePasswordSchema.safeParse({
        current_password: 'Зарядка2026',
        new_password: 'Зарядка2026',
      }).error?.issues[0]?.path,
    ).toEqual(['new_password']);
  });
});

describe('reportSchema', () => {
  it('для «другое» требует пояснение', () => {
    expect(reportSchema.safeParse({ reason: 'spam' }).success).toBe(true);
    expect(
      reportSchema.safeParse({ reason: 'other' }).error?.issues[0]?.path,
    ).toEqual(['details']);
    expect(
      reportSchema.safeParse({ reason: 'other', details: 'Фейк' }).success,
    ).toBe(true);
  });
});
