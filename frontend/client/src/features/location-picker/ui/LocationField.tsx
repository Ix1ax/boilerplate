import { useState } from 'react';
import { MapPinIcon } from 'lucide-react';

import { Button } from '@/shared/ui';

import { LocationPickerDialog } from './LocationPickerDialog';

interface LocationFieldProps {
  lat: number | null | undefined;
  lng: number | null | undefined;
  onChange: (value: {
    lat: number;
    lng: number;
    address?: string;
  }) => void;
}

/** Кнопка рядом с полем «Место и адрес»: ставит lat/lng точкой на карте, адрес — опционально. */
export const LocationField = ({ lat, lng, onChange }: LocationFieldProps) => {
  const [open, setOpen] = useState(false);
  const hasPoint = lat != null && lng != null;

  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
      >
        <MapPinIcon aria-hidden />
        {hasPoint ? 'Точка на карте указана' : 'Указать точку на карте'}
      </Button>
      <LocationPickerDialog
        open={open}
        onOpenChange={setOpen}
        initial={{ lat: lat ?? null, lng: lng ?? null }}
        onConfirm={({ lat: nextLat, lng: nextLng, address }) =>
          onChange({
            lat: nextLat,
            lng: nextLng,
            address: address || undefined,
          })
        }
      />
    </>
  );
};
