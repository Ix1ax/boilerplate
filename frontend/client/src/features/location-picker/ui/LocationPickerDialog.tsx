import { useState } from 'react';
import { SearchIcon } from 'lucide-react';

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
} from '@/shared/ui';

import { useNominatimSearch } from '../model';
import { LocationMap } from './LocationMap';

interface LocationPickerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initial: { lat: number | null; lng: number | null };
  onConfirm: (value: { lat: number; lng: number; address: string }) => void;
}

/** Поиск адреса (Nominatim/OSM) и установка точки кликом по карте или перетаскиванием маркера. */
export const LocationPickerDialog = ({
  open,
  onOpenChange,
  initial,
  onConfirm,
}: LocationPickerDialogProps) => {
  const [query, setQuery] = useState('');
  const [point, setPoint] = useState(initial);
  const [flyTo, setFlyTo] = useState<{ lat: number; lng: number } | null>(
    null,
  );
  const [address, setAddress] = useState('');
  const search = useNominatimSearch(query);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Точка на карте</DialogTitle>
          <DialogDescription>
            Найдите адрес или кликните по карте. Маркер можно перетащить.
          </DialogDescription>
        </DialogHeader>

        <div className="relative">
          <SearchIcon
            aria-hidden
            className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Улица, дом, площадка…"
            className="pl-9"
          />
          {search.data && search.data.length > 0 && (
            <ul className="absolute z-10 mt-1 w-full overflow-hidden rounded-xl bg-popover shadow-md ring-1 ring-foreground/10">
              {search.data.map((result) => (
                <li key={`${result.lat},${result.lon}`}>
                  <button
                    type="button"
                    className="block w-full px-3 py-2 text-left text-sm hover:bg-accent"
                    onClick={() => {
                      const lat = Number(result.lat);
                      const lng = Number(result.lon);
                      setPoint({ lat, lng });
                      setFlyTo({ lat, lng });
                      setAddress(result.display_name);
                      setQuery(result.display_name);
                    }}
                  >
                    {result.display_name}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <LocationMap
          lat={point.lat}
          lng={point.lng}
          flyTo={flyTo}
          onPick={(lat, lng) => {
            setPoint({ lat, lng });
            setAddress('');
          }}
        />

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Отмена
          </Button>
          <Button
            disabled={point.lat == null || point.lng == null}
            onClick={() => {
              if (point.lat == null || point.lng == null) return;
              onConfirm({
                lat: point.lat,
                lng: point.lng,
                address: address || query,
              });
              onOpenChange(false);
            }}
          >
            Использовать точку
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
