import { Button } from '@/shared/ui';

import { useLogin } from '../model';
export const LoginForm = () => {
  const login = useLogin();
  return (
    <Button className="w-full" onClick={login}>
      Войти или зарегистрироваться
    </Button>
  );
};
