import { useEffect, useRef } from 'react';
import L from 'leaflet';
import markerIcon from 'leaflet/dist/images/marker-icon.png';
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import markerShadow from 'leaflet/dist/images/marker-shadow.png';

import 'leaflet/dist/leaflet.css';

// Vite отдаёт изображения по хешированным URL — иконку маркера по умолчанию нужно переопределить явно.
L.Icon.Default.mergeOptions({
  iconUrl: markerIcon,
  iconRetinaUrl: markerIcon2x,
  shadowUrl: markerShadow,
});

const KRASNOYARSK: [number, number] = [56.0097, 92.8673];

interface LocationMapProps {
  lat: number | null;
  lng: number | null;
  onPick: (lat: number, lng: number) => void;
  /** Координаты, на которые нужно перелететь (клик по результату поиска). */
  flyTo?: { lat: number; lng: number } | null;
}

export const LocationMap = ({ lat, lng, onPick, flyTo }: LocationMapProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<L.Map | null>(null);
  const markerRef = useRef<L.Marker | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const map = L.map(containerRef.current, {
      center: lat != null && lng != null ? [lat, lng] : KRASNOYARSK,
      zoom: lat != null && lng != null ? 15 : 12,
    });
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution:
        '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      maxZoom: 19,
    }).addTo(map);
    map.on('click', (event: L.LeafletMouseEvent) => {
      onPick(event.latlng.lat, event.latlng.lng);
    });
    mapRef.current = map;
    const timer = setTimeout(() => map.invalidateSize(), 80);
    return () => {
      clearTimeout(timer);
      map.remove();
      mapRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    if (lat == null || lng == null) {
      markerRef.current?.remove();
      markerRef.current = null;
      return;
    }
    if (markerRef.current) {
      markerRef.current.setLatLng([lat, lng]);
    } else {
      markerRef.current = L.marker([lat, lng], { draggable: true })
        .addTo(map)
        .on('dragend', (event) => {
          const position = (event.target as L.Marker).getLatLng();
          onPick(position.lat, position.lng);
        });
    }
  }, [lat, lng, onPick]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || !flyTo) return;
    map.flyTo([flyTo.lat, flyTo.lng], 16);
  }, [flyTo]);

  return (
    <div
      ref={containerRef}
      className="h-[360px] w-full overflow-hidden rounded-2xl"
    />
  );
};
