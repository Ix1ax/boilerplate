import { useEffect, useRef } from 'react';
import L from 'leaflet';

import type { Event } from '@/entities/event';

import 'leaflet/dist/leaflet.css';

interface EventMapProps {
  events: Event[];
  selected: string | null;
  onSelect: (id: string) => void;
  onBounds: (bbox: string) => void;
  center: [number, number] | null;
}

export const EventMap = ({
  events,
  selected,
  onSelect,
  onBounds,
  center,
}: EventMapProps) => {
  const container = useRef<HTMLDivElement>(null);
  const map = useRef<L.Map | null>(null);
  const callbacks = useRef({ onSelect, onBounds });
  useEffect(() => {
    callbacks.current = { onSelect, onBounds };
  }, [onSelect, onBounds]);
  useEffect(() => {
    if (!container.current) return;
    const instance = L.map(container.current, {
      center: [56.0097, 92.8673],
      zoom: 12,
    });
    map.current = instance;
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution:
        '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      maxZoom: 19,
    }).addTo(instance);
    const update = () => {
      const b = instance.getBounds();
      callbacks.current.onBounds(
        [b.getWest(), b.getSouth(), b.getEast(), b.getNorth()].join(','),
      );
    };
    instance.on('moveend', update);
    instance.whenReady(update);
    return () => {
      instance.remove();
      map.current = null;
    };
  }, []);
  useEffect(() => {
    if (!map.current) return;
    const layer = L.layerGroup().addTo(map.current);
    const groups = new Map<string, Event[]>();
    events.forEach((event) => {
      if (event.lat == null || event.lng == null) return;
      const key = `${event.lat.toFixed(4)},${event.lng.toFixed(4)}:${event.location}`;
      groups.set(key, [...(groups.get(key) ?? []), event]);
    });
    groups.forEach((group) => {
      const event = group[0];
      const active = group.some((item) => item.id === selected);
      const content = document.createElement('span');
      content.textContent = String(group.length);
      content.className = `grid size-10 place-items-center rounded-full border-2 border-white font-bold shadow-md ${active ? 'bg-gold text-ink' : 'bg-blue text-white'}`;
      L.marker([event.lat!, event.lng!], {
        icon: L.divIcon({
          html: content,
          className: '',
          iconSize: [40, 40],
          iconAnchor: [20, 20],
        }),
        title: `${event.location}: встреч ${group.length}`,
        keyboard: true,
      })
        .on('click', () => callbacks.current.onSelect(event.id))
        .addTo(layer);
    });
    return () => {
      layer.remove();
    };
  }, [events, selected]);
  useEffect(() => {
    if (center) map.current?.flyTo(center, 14);
  }, [center]);
  return (
    <div
      ref={container}
      className="relative z-0 h-[520px] w-full rounded-3xl"
      aria-label="Карта площадок и встреч OpenStreetMap"
    />
  );
};
