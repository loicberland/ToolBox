import { useCallback, useLayoutEffect, useRef } from 'react';
import type { RefCallback } from 'react';

type ItemId = string | number;
type RectMap<TId extends ItemId> = Map<TId, DOMRect>;

const transition = 'transform 220ms cubic-bezier(0.2, 0, 0, 1)';
const durationMs = 240;

export function useFlipReorderAnimation<TId extends ItemId>() {
  const itemRefs = useRef(new Map<TId, HTMLElement>());
  const firstRects = useRef<RectMap<TId> | undefined>();

  const registerItem = useCallback((id: TId): RefCallback<HTMLElement> => {
    return (node) => {
      if (node) {
        itemRefs.current.set(id, node);
      } else {
        itemRefs.current.delete(id);
      }
    };
  }, []);

  const animateReorder = useCallback((callback: () => void) => {
    const rects: RectMap<TId> = new Map();
    itemRefs.current.forEach((node, id) => {
      rects.set(id, node.getBoundingClientRect());
    });
    firstRects.current = rects;
    callback();
  }, []);

  useLayoutEffect(() => {
    const rects = firstRects.current;
    if (!rects) {
      return;
    }
    firstRects.current = undefined;

    const animatedNodes: HTMLElement[] = [];

    itemRefs.current.forEach((node, id) => {
      const first = rects.get(id);
      if (!first) {
        return;
      }
      const last = node.getBoundingClientRect();
      const deltaY = first.top - last.top;
      if (Math.abs(deltaY) < 1) {
        return;
      }

      node.style.transition = 'none';
      node.style.transform = `translateY(${deltaY}px)`;
      node.style.zIndex = '1';
      animatedNodes.push(node);
    });

    if (animatedNodes.length === 0) {
      return;
    }

    window.requestAnimationFrame(() => {
      animatedNodes.forEach((node) => {
        node.style.transition = transition;
        node.style.transform = 'translateY(0)';
      });
    });

    window.setTimeout(() => {
      animatedNodes.forEach((node) => {
        node.style.transition = '';
        node.style.transform = '';
        node.style.zIndex = '';
      });
    }, durationMs);
  });

  return { registerItem, animateReorder };
}
