import { Link } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';
import { Button, RadarRings } from '@/shared/ui';

export const ForbiddenPage = () => <section className="relative mx-auto grid min-h-[65vh] w-full max-w-5xl content-center justify-items-start gap-5 overflow-hidden px-5 py-16 sm:px-8"><RadarRings tone="blue" sizes={[160,300,440]} strength={0.2} className="top-1/2 right-10"/><span className="relative text-7xl font-extrabold text-blue">403</span><h1 className="title-display relative text-4xl">Здесь нужны другие права</h1><p className="relative max-w-xl text-lg">Этот раздел доступен организатору или администратору. Проверьте, под каким аккаунтом вы вошли.</p><div className="relative flex flex-wrap gap-3"><Button render={<Link to={STATIC_ROUTES.EVENTS}/>}>Найти встречу</Button><Button variant="outline" render={<Link to={STATIC_ROUTES.LOGIN}/>}>Перейти ко входу</Button></div></section>;
