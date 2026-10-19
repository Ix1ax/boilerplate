// Контракты кейса «Платформа совместных спортивных активностей»: типы ответов (из openapi.yaml)
// и Zod-схемы форм. Правила схем повторяют проверки сервера (server/events/internal/events):
// клиент показывает ошибку сразу, сервер всё равно проверяет сам и отвечает 422 с полем field.
import { z } from 'zod';

import type { components } from './api.gen';

type Schemas = components['schemas'];

// ── Типы ответов ────────────────────────────────────────────────────────────

export type Sport = Schemas['Sport'];
export type AdminSport = Schemas['AdminSport'];
export type SportEvent = Schemas['Event'];
export type SportEventPage = Schemas['EventPage'];
export type EventStatus = SportEvent['status'];
export type JoinMode = SportEvent['join_mode'];
export type QuorumStatus = SportEvent['quorum_status'];
export type Organizer = Schemas['Organizer'];
export type MyParticipation = Schemas['MyParticipation'];
export type ParticipationStatus = MyParticipation['status'];
export type Participant = Schemas['Participant'];
export type MyEntry = Schemas['MyEntry'];
export type Person = Schemas['Person'];
export type EventComment = Schemas['Comment'];
export type Organization = Schemas['Organization'];
export type OrganizationStatus = Organization['status'];
export type OrganizationPage = Schemas['OrganizationPage'];
export type OrganizationDetail = Schemas['OrganizationDetail'];
export type Profile = Schemas['Profile'];
export type OrganizerStats = Schemas['OrganizerStats'];
export type CompanyRequest = Schemas['CompanyRequest'];
export type Demand = Schemas['Demand'];
export type EventWeather = Schemas['EventWeather'];
export type Transition = Schemas['Transition'];
export type HistoryEntry = Schemas['HistoryEntry'];
export type ModerationDashboard = Schemas['Dashboard'];
export type ModerationLogEntry = Schemas['LogEntry'];
export type ModerationLogPage = Schemas['LogPage'];
export type ModerationComment = Schemas['ModerationComment'];
export type ModerationCommentPage = Schemas['CommentPage'];

export type Report = Schemas['Report'];
export type ReportPage = Schemas['ReportPage'];
export type MyReport = Schemas['MyReport'];
export type ReportReason = Schemas['ReportInput']['reason'];
export type ReportStatus = MyReport['status'];
export type ReportTargetType = Report['target_type'];
export type ReportAnalysis = Schemas['ReportAnalysis'];
export type ReportAnalysisFlag = ReportAnalysis['flags'][number]['code'];
export type Account = Schemas['Account'];
export type AccountPage = Schemas['AccountPage'];
export type AccountDetail = Schemas['AccountDetail'];
/** Администратор текущей админской сессии (GET /api/v1/admin/me) — отдельный вход, не CurrentUser. */
export type AdminUser = Schemas['AdminUser'];
/** Учётная запись администратора (управляет суперадминистратор: /api/v1/admin/admins). */
// Имена Administrator*, а не AdminAccount*: AdminAccount в admin.ts — учётная запись пользователя в админке.
export type Administrator = Schemas['Admin'];
export type AdministratorPage = Schemas['AdminPage'];
export type AdministratorDetail = Schemas['AdminDetail'];
export type AdminRole = Administrator['role'];
export type AdminStatus = Administrator['status'];
/** Текущий пользователь (GET /api/v1/me); provider — способ входа. */
export type CurrentUser = Schemas['User'];
export type AuthProvider = CurrentUser['provider'];

// ── Подписи для интерфейса ─────────────────────────────────────────────────
// Одни и те же слова на всех экранах: человек не должен гадать, «ожидает» и «на рассмотрении» — это одно и то же или нет.

export const levelLabels = {
  any: 'Для всех',
  beginner: 'Можно впервые',
  amateur: 'Любителям',
  pro: 'Опытным',
} as const;

export const joinModeLabels: Record<JoinMode, string> = {
  open: 'Свободная запись',
  request: 'По заявке',
};

export const participationLabels: Record<ParticipationStatus, string> = {
  confirmed: 'Вы участвуете',
  pending: 'Заявка на рассмотрении',
  // rejected — и отказ по заявке, и исключение организатором: в обоих случаях повторная запись закрыта.
  rejected: 'Организатор отказал в участии',
  cancelled: 'Вы отменили запись',
  waitlist: 'Вы в листе ожидания',
};

export const eventStatusLabels: Record<EventStatus, string> = {
  published: 'Опубликовано',
  unpublished: 'Снято с публикации',
  deleted: 'Удалено модератором',
  cancelled: 'Отменено',
};

export const organizationStatusLabels: Record<OrganizationStatus, string> = {
  pending: 'На рассмотрении',
  approved: 'Проверенная организация',
  rejected: 'Заявка отклонена',
};

export const reportReasonLabels: Record<ReportReason, string> = {
  spam: 'Спам или реклама',
  offensive: 'Оскорбления',
  fraud: 'Мошенничество',
  wrong_info: 'Неверные сведения: место, время, условия',
  dangerous: 'Угроза безопасности',
  other: 'Другое',
};

export const reportStatusLabels: Record<ReportStatus, string> = {
  open: 'На рассмотрении',
  resolved: 'Меры приняты',
  dismissed: 'Нарушений не нашли',
};

/** На что жалоба — подпись для очереди жалоб в админке и для «моих жалоб». */
export const reportTargetLabels: Record<ReportTargetType, string> = {
  event: 'Мероприятие',
  comment: 'Сообщение в обсуждении',
  review: 'Отзыв',
  story: 'История',
  direct_message: 'Личное сообщение',
};

/** Подсказка ИИ по жалобе: решает админ, подписи не должны звучать как приговор. */
export const aiVerdictLabels: Record<ReportAnalysis['verdict'], string> = {
  ok: 'Признаков нарушения не видно',
  needs_review: 'Есть признаки нарушения — проверьте',
};

export const aiRiskLabels: Record<
  NonNullable<ReportAnalysis['advice']>['risk'],
  string
> = {
  low: 'Низкий риск',
  medium: 'Средний риск',
  high: 'Высокий риск',
};

export const aiFlagLabels: Record<ReportAnalysisFlag, string> = {
  money: 'Деньги или реквизиты',
  commerce: 'Реклама или продажа',
  insult: 'Оскорбления',
  link: 'Ссылка',
  contact: 'Телефон или e-mail',
  shouting: 'Заглавные буквы',
  many_reports: 'Много жалоб',
};

export const authProviderLabels: Record<AuthProvider, string> = {
  sso: 'Столото ID',
  local: 'Почта и пароль',
};

export const quorumLabels: Record<QuorumStatus, string> = {
  none: '',
  gathering: 'Собираем состав',
  reached: 'Состав собран — встреча состоится',
  failed: 'Состав не собран',
};

export const dayPartLabels = {
  morning: 'Утро (до 12:00)',
  day: 'День (12–17)',
  evening: 'Вечер (после 17)',
} as const;

export const weekdayLabels = [
  'Пн',
  'Вт',
  'Ср',
  'Чт',
  'Пт',
  'Сб',
  'Вс',
] as const;

// ── Состояние главной кнопки на карточке события ───────────────────────────

export type JoinAction =
  | 'join' // Записаться (свободная запись)
  | 'apply' // Подать заявку
  | 'manage' // Вы организатор → «Управлять»
  | 'leave' // Вы участвуете / в листе ожидания / заявка ждёт → можно отменить
  | 'rejoin' // Сами отменили запись → можно снова
  | 'refused' // Организатор отказал или исключил → повторно нельзя (сервер ответит 409): показать причину и «другие встречи»
  | 'waitlist' // Мест нет → встать в лист ожидания (свободная запись)
  | 'closed'; // Снято, отменено или уже началось — записи нет

/**
 * Что показать на главной кнопке. Сервер всё равно проверяет запись сам (409 на снятое и прошедшее),
 * но кнопка не должна предлагать заведомо невозможное.
 */
export function joinAction(e: SportEvent, now: Date = new Date()): JoinAction {
  if (e.is_organizer) return 'manage';
  const ended = new Date(e.ends_at ?? e.starts_at) <= now;
  if (e.status !== 'published' || ended) return 'closed';
  const mine = e.my_participation?.status;
  if (mine === 'confirmed' || mine === 'pending' || mine === 'waitlist')
    return 'leave';
  if (mine === 'rejected') return 'refused';
  if (mine === 'cancelled') return 'rejoin';
  if (e.join_mode === 'request') return 'apply';
  return e.spots_left === 0 ? 'waitlist' : 'join';
}

// ── Схемы форм ─────────────────────────────────────────────────────────────

/** Длина в символах, а не в UTF-16: эмодзи и буквы считаются одинаково, как на сервере. */
const chars = (min: number, max: number, message: string) =>
  z
    .string()
    .trim()
    .refine((v) => {
      const n = Array.from(v).length;
      return n >= min && n <= max;
    }, message);

const optionalText = (max: number, message: string) =>
  chars(0, max, message).default('');
const uuid = z.uuid('Неизвестный идентификатор');
const isoDate = z.iso.datetime({
  offset: true,
  message: 'Укажите дату и время',
});
const level = z.enum(['any', 'beginner', 'amateur', 'pro']);
const lat = z.number().min(-90).max(90).nullable().optional();
const lng = z.number().min(-180).max(180).nullable().optional();

const bothOrNone = (v: { lat?: number | null; lng?: number | null }) =>
  (v.lat == null) === (v.lng == null);

/** Направление: из справочника (sport_id) или своё (sport_name) — новое заводится без правки кода. */
const sportChoice = {
  sport_id: uuid.optional(),
  sport_name: z
    .string()
    .transform((v) => v.trim().split(/\s+/).join(' '))
    .optional(),
};
const hasSport = (v: { sport_id?: string; sport_name?: string }) =>
  Boolean(v.sport_id) ||
  (v.sport_name !== undefined &&
    Array.from(v.sport_name).length >= 2 &&
    Array.from(v.sport_name).length <= 60);

// Поля события без значений по умолчанию: их же использует правка, где непереданное поле не трогаем.
const eventFields = {
  zone: chars(0, 100, 'Зона — не длиннее 100 символов').optional(),
  meeting_host: chars(
    0,
    200,
    'Кто встретит — не длиннее 200 символов',
  ).optional(),
  step_free: z.boolean().optional(),
  companion_allowed: z.boolean().optional(),
  title: chars(3, 120, 'Название — от 3 до 120 символов'),
  description: chars(0, 5000, 'Описание — не длиннее 5000 символов'),
  starts_at: isoDate,
  ends_at: isoDate.nullable().optional(),
  location: chars(3, 300, 'Укажите адрес или понятное описание места'),
  lat,
  lng,
  level,
  requirements: chars(0, 2000, 'Условия — не длиннее 2000 символов'),
  price_note: chars(0, 200, 'Стоимость — не длиннее 200 символов'),
  private_info: chars(0, 2000, 'Точка сбора — не длиннее 2000 символов'),
  capacity: z
    .number()
    .int()
    .min(1, 'Хотя бы одно место')
    .max(10000)
    .nullable()
    .optional(),
  outdoor: z.boolean(),
};

/** Создание события (POST /events). */
export const eventInputSchema = z
  .object({
    ...eventFields,
    format: z.enum(['offline', 'online']).default('offline'),
    zone: eventFields.zone.default(''),
    meeting_host: eventFields.meeting_host.default(''),
    step_free: eventFields.step_free.default(false),
    companion_allowed: eventFields.companion_allowed.default(false),
    description: eventFields.description.default(''),
    level: level.default('any'),
    requirements: eventFields.requirements.default(''),
    price_note: eventFields.price_note.default(''),
    private_info: eventFields.private_info.default(''),
    outdoor: z.boolean().default(false),
    ...sportChoice,
    join_mode: z.enum(
      ['open', 'request'],
      'Выберите режим записи: свободно или по заявке',
    ),
    as_organization: z.boolean().default(false),
    min_participants: z
      .number()
      .int()
      .min(2, 'Минимум — от 2 участников')
      .max(1000)
      .nullable()
      .optional(),
    quorum_deadline: isoDate.nullable().optional(),
  })
  .refine(hasSport, {
    path: ['sport'],
    message: 'Выберите направление или впишите своё',
  })
  .refine((v) => new Date(v.starts_at) > new Date(), {
    path: ['starts_at'],
    message: 'Укажите дату и время в будущем',
  })
  .refine((v) => !v.ends_at || new Date(v.ends_at) > new Date(v.starts_at), {
    path: ['ends_at'],
    message: 'Окончание должно быть позже начала',
  })
  .refine(bothOrNone, {
    path: ['location'],
    message: 'Отметьте точку на карте целиком',
  })
  .refine(
    (v) =>
      v.min_participants == null ||
      v.capacity == null ||
      v.min_participants <= v.capacity,
    {
      path: ['min_participants'],
      message: 'Минимум не может быть больше числа мест',
    },
  )
  .refine((v) => v.quorum_deadline == null || v.min_participants != null, {
    path: ['min_participants'],
    message: 'Срок сбора состава задаётся вместе с минимумом участников',
  })
  .refine(
    (v) => !v.quorum_deadline || new Date(v.quorum_deadline) > new Date(),
    {
      path: ['quorum_deadline'],
      message: 'Срок сбора состава должен быть в будущем',
    },
  )
  .refine(
    (v) =>
      !v.quorum_deadline ||
      new Date(v.quorum_deadline) <= new Date(v.starts_at),
    {
      path: ['quorum_deadline'],
      message: 'Состав нужно собрать до начала мероприятия',
    },
  );
export type EventFormInput = z.input<typeof eventInputSchema>;
export type EventFormValues = z.output<typeof eventInputSchema>;

/** Правка события организатором (PATCH /events/{id}): только меняемые поля. */
export const eventPatchSchema = z
  .object(eventFields)
  .partial()
  .refine((v) => !v.starts_at || new Date(v.starts_at) > new Date(), {
    path: ['starts_at'],
    message: 'Укажите дату и время в будущем',
  })
  .refine(bothOrNone, {
    path: ['location'],
    message: 'Отметьте точку на карте целиком',
  });
/** Форма правки всегда содержит текущее начало; очистка не должна молча исчезать из PATCH. */
export const eventEditSchema = eventPatchSchema
  .refine((v) => !!v.starts_at, {
    path: ['starts_at'],
    message: 'Укажите дату и время начала',
  })
  .refine(
    (v) =>
      !v.starts_at || !v.ends_at || new Date(v.ends_at) > new Date(v.starts_at),
    {
      path: ['ends_at'],
      message: 'Окончание должно быть позже начала',
    },
  );
export type EventPatchValues = z.output<typeof eventPatchSchema>;

/** Запись или заявка (POST /events/{id}/join). */
export const joinSchema = z.object({
  message: optionalText(500, 'Сообщение к заявке — не длиннее 500 символов'),
  first_time: z.boolean().default(false),
});
export type JoinValues = z.output<typeof joinSchema>;

/** Вопрос или ответ в обсуждении. */
export const commentSchema = z.object({
  body: chars(1, 2000, 'Сообщение — от 1 до 2000 символов'),
  parent_id: uuid.optional(),
});
export type CommentValues = z.output<typeof commentSchema>;

/** Причина решения, которую увидит другая сторона: отказ организации, снятие события, скрытие сообщения. */
export const reasonSchema = z.object({
  reason: chars(1, 1000, 'Укажите причину: её увидит другая сторона'),
});
/** Причина решения по отзыву, истории или личному сообщению: уходит автору, сервер ограничивает 500 символами. */
export const socialModerationReasonSchema = z.object({
  reason: chars(1, 500, 'Укажите причину до 500 символов: её увидит автор'),
});
/** Необязательная причина: отказ в заявке участника, отмена события организатором — тоже обязательна на сервере. */
export const optionalReasonSchema = z.object({
  reason: optionalText(1000, 'Причина — не длиннее 1000 символов'),
});

/** Заявка или правка организации (POST/PUT /my/organization). */
export const organizationInputSchema = z.object({
  name: chars(2, 120, 'Название — от 2 до 120 символов'),
  description: chars(
    10,
    3000,
    'Расскажите о деятельности: от 10 до 3000 символов',
  ),
  city: optionalText(100, 'Город — не длиннее 100 символов'),
  contacts: optionalText(500, 'Контакты — не длиннее 500 символов'),
  sport_ids: z.array(uuid).max(20, 'Не больше 20 направлений').default([]),
});
export type OrganizationValues = z.output<typeof organizationInputSchema>;

/** «Ищу компанию» (POST /my/company-requests). Пустые списки времени и дней — «когда угодно». */
export const companyRequestSchema = z
  .object({
    ...sportChoice,
    level: level.default('any'),
    day_parts: z.array(z.enum(['morning', 'day', 'evening'])).default([]),
    weekdays: z.array(z.number().int().min(1).max(7)).default([]),
    area: optionalText(200, 'Район — не длиннее 200 символов'),
    lat,
    lng,
    radius_km: z
      .number()
      .int()
      .min(1)
      .max(100, 'Радиус — до 100 км')
      .nullable()
      .optional(),
    note: optionalText(500, 'Комментарий — не длиннее 500 символов'),
  })
  .refine(hasSport, {
    path: ['sport'],
    message: 'Выберите направление или впишите своё',
  })
  .refine(bothOrNone, {
    path: ['location'],
    message: 'Отметьте точку на карте целиком',
  })
  .refine((v) => v.radius_km == null || v.lat != null, {
    path: ['radius_km'],
    message: 'Радиус задаётся вместе с точкой на карте',
  });
export type CompanyRequestValues = z.output<typeof companyRequestSchema>;

/** Мой профиль (PATCH /my/profile). Пустой уровень — «не указывать». */
// Без значений по умолчанию: непереданное поле сервер не трогает, а '' по умолчанию стёрло бы его.
export const profilePatchSchema = z.object({
  about: chars(0, 1000, 'О себе — не длиннее 1000 символов').optional(),
  city: chars(0, 100, 'Город — не длиннее 100 символов').optional(),
  level: z.enum(['', 'beginner', 'amateur', 'pro']).optional(),
});
export type ProfilePatchValues = z.output<typeof profilePatchSchema>;

/** Жалоба на мероприятие, сообщение, отзыв, историю или личное сообщение. Для «Другое» пояснение обязательно, как на сервере. */
export const reportSchema = z
  .object({
    reason: z.enum(
      ['spam', 'offensive', 'fraud', 'wrong_info', 'dangerous', 'other'],
      'Выберите причину жалобы',
    ),
    details: optionalText(1000, 'Пояснение — не длиннее 1000 символов'),
  })
  .refine((v) => v.reason !== 'other' || v.details !== '', {
    path: ['details'],
    message: 'Опишите, что не так: для причины «другое» пояснение обязательно',
  });
export type ReportValues = z.output<typeof reportSchema>;

/** Отклонение жалобы админом: пояснение увидит её автор. */
export const reportResolutionSchema = z.object({
  resolution: chars(1, 1000, 'Поясните решение: его увидит автор жалобы'),
});

// ── Вход по почте и паролю (сервис identity) ─────────────────────────────────
// Правила пароля повторяют identity: 8–128 символов, буквы и цифры. Список частых паролей
// проверяет только сервер — клиент узнает о нём по 422 с field=password.

const email = z
  .string()
  .trim()
  .toLowerCase()
  .pipe(z.email('Укажите почту в формате name@example.ru').max(254));
const password = chars(8, 128, 'Пароль — от 8 до 128 символов').refine(
  (v) => /\p{L}/u.test(v) && /\p{N}/u.test(v),
  'Пароль должен содержать буквы и цифры',
);

/** Регистрация (POST /auth/local/register). */
export const registerSchema = z.object({
  email,
  password,
  name: chars(2, 100, 'Имя — от 2 до 100 символов'),
});
export type RegisterValues = z.output<typeof registerSchema>;

/** Вход (POST /auth/local/login): формат пароля не проверяем — иначе подсказали бы правила старым пользователям. */
export const loginSchema = z.object({
  email,
  password: z.string().min(1, 'Введите пароль'),
});
export type LoginValues = z.output<typeof loginSchema>;

/** Смена пароля (POST /auth/local/password). */
export const changePasswordSchema = z
  .object({
    current_password: z.string().min(1, 'Введите текущий пароль'),
    new_password: password,
  })
  .refine((v) => v.current_password !== v.new_password, {
    path: ['new_password'],
    message: 'Новый пароль совпадает с текущим',
  });
export type ChangePasswordValues = z.output<typeof changePasswordSchema>;

// ── Вход в админку (отдельные учётные записи администраторов) ───────────────
// Пароль администратора длиннее: от 12 символов. Сервер проверяет то же самое и список частых паролей.

const adminPassword = chars(
  12,
  128,
  'Пароль администратора — от 12 до 128 символов',
).refine(
  (v) => /\p{L}/u.test(v) && /\p{N}/u.test(v),
  'Пароль должен содержать буквы и цифры',
);

/** Вход в админку (POST /auth/admin/login). */
export const adminLoginSchema = loginSchema;
export type AdminLoginValues = LoginValues;

/** Смена своего пароля администратора (POST /auth/admin/password). */
export const adminChangePasswordSchema = z
  .object({
    current_password: z.string().min(1, 'Введите текущий пароль'),
    new_password: adminPassword,
  })
  .refine((v) => v.current_password !== v.new_password, {
    path: ['new_password'],
    message: 'Новый пароль совпадает с текущим',
  });
export type AdminChangePasswordValues = z.output<
  typeof adminChangePasswordSchema
>;

/** Новый администратор (POST /api/v1/admin/admins) — только суперадминистратор. */
export const adminCreateSchema = z.object({
  email,
  name: chars(2, 100, 'Имя — от 2 до 100 символов'),
  password: adminPassword,
});
export type AdminCreateValues = z.output<typeof adminCreateSchema>;

/** Новый пароль администратору (POST /api/v1/admin/admins/{id}/password). */
export const adminResetPasswordSchema = z.object({ password: adminPassword });
export type AdminResetPasswordValues = z.output<
  typeof adminResetPasswordSchema
>;

export const adminRoleLabels: Record<AdminRole, string> = {
  superadmin: 'Суперадминистратор',
  admin: 'Администратор',
};

export const adminStatusLabels: Record<AdminStatus, string> = {
  active: 'Активен',
  blocked: 'Заблокирован',
  deleted: 'Удалён',
};
