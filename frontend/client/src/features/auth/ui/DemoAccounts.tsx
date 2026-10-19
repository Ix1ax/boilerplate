import { Button } from '@/shared/ui';

import { useDemoAccounts, useDemoLogin, useLoginTarget } from '../model';

/** «Демодоступы для жюри»: обычные учётные записи с выданными правами, вход в одно нажатие. */
export const DemoAccounts = () => {
  const target = useLoginTarget();
  const { data: accounts = [] } = useDemoAccounts();
  const demo = useDemoLogin(target);

  if (!accounts.length) return null;

  return (
    <div className="flex flex-col gap-2.5 rounded-3xl border-2 border-dashed border-foreground bg-card p-[22px]">
      <b className="title-display text-lg">Демодоступы для жюри</b>
      <span className="text-[15px] text-ink-soft dark:text-muted-foreground">
        Обычные учётные записи с выданными правами, а не переключатель ролей.
        Нажмите — и войдёте под выбранным аккаунтом.
      </span>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(220px,100%),1fr))] gap-2">
        {accounts.map((account) => (
          <button
            key={account.key}
            type="button"
            title={account.hint}
            disabled={demo.isPending}
            onClick={() => demo.mutate(account.key)}
            className="flex min-h-[60px] flex-col justify-center rounded-2xl border-[1.5px] border-line-strong bg-background px-3.5 py-2 text-left text-[15px] transition-colors hover:border-foreground disabled:opacity-60 dark:border-border"
          >
            <b className="text-base">
              {demo.isPending && demo.variables === account.key
                ? 'Входим…'
                : account.name}
            </b>
            <span>{account.role}</span>
          </button>
        ))}
      </div>
      {demo.isError && (
        <div
          role="alert"
          className="flex flex-wrap items-center gap-3 rounded-[14px] bg-danger-soft px-4 py-3.5 font-semibold text-danger"
        >
          {demo.error.message}
          <Button variant="outline" size="xs" onClick={() => demo.reset()}>
            Понятно
          </Button>
        </div>
      )}
    </div>
  );
};
