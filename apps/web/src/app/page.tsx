import { Dashboard } from '@/screens/dashboard/Dashboard';
export default async function Page({
  searchParams,
}: {
  searchParams: Promise<{ auth_error?: string }>;
}) {
  const params = await searchParams;
  return (
    <>
      {params.auth_error && (
        <p role="alert" className="bg-destructive p-3 text-center text-white">
          Не удалось войти. Попробуйте ещё раз.
        </p>
      )}
      <Dashboard />
    </>
  );
}
