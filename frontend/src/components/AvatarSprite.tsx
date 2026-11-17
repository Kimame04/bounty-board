import React, { useMemo } from 'react';

interface AvatarSpriteProps {
  username: string;
  level?: number;
  size?: number; // Visual display size in pixels (e.g. 40, 64)
}

// Simple deterministic hash from string
function hashString(str: string): number[] {
  const bytes: number[] = [];
  let h1 = 0xdeadbeef, h2 = 0x41c64e6d;
  for (let i = 0; i < str.length; i++) {
    const ch = str.charCodeAt(i);
    h1 = Math.imul(h1 ^ ch, 2654435761);
    h2 = Math.imul(h2 ^ ch, 1597334677);
  }
  h1 = Math.imul(h1 ^ (h1 >>> 16), 2246822507);
  h1 ^= Math.imul(h2 ^ (h2 >>> 13), 3266489909);
  h2 = Math.imul(h2 ^ (h2 >>> 16), 2246822507);
  h2 ^= Math.imul(h1 ^ (h1 >>> 13), 3266489909);

  for (let i = 0; i < 16; i++) {
    bytes.push(((h1 >>> (i * 2)) ^ (h2 >>> (i * 3))) & 0xff);
  }
  return bytes;
}

// Palettes for adventurer aesthetics
const SKIN_TONES = ['#ffd1b3', '#fcd5b5', '#e0ac69', '#c68642', '#8d5524', '#ffdbac'];
const HAIR_COLORS = ['#8d4a2b', '#2c222b', '#d6a054', '#b55239', '#5a3825', '#e6e6e6'];
const ARMOR_COLORS = [
  '#4a6fa5', // Cobalt
  '#598234', // Forest Ranger
  '#993333', // Crimson Knight
  '#6b5b95', // Mystic Purple
  '#d4af37', // Guild Gold
  '#36454f', // Shadow Iron
];

export const AvatarSprite: React.FC<AvatarSpriteProps> = ({ username, level = 1, size = 48 }) => {
  const spriteData = useMemo(() => {
    const h = hashString(username.toLowerCase());
    const skin = SKIN_TONES[h[0] % SKIN_TONES.length];
    const hair = HAIR_COLORS[h[1] % HAIR_COLORS.length];
    const armor = ARMOR_COLORS[h[2] % ARMOR_COLORS.length];
    const eyeColor = '#1c1c1c';

    // 8x8 Grid representing character body
    // 0: empty, 1: hair/helm, 2: skin, 3: eyes, 4: armor/tunic, 5: weapon/shield/accessory
    const grid: number[][] = Array.from({ length: 8 }, () => Array(8).fill(0));

    // Head / Hair (Rows 1 to 3)
    for (let r = 1; r <= 3; r++) {
      for (let c = 2; c <= 3; c++) {
        // Mirrored horizontally: col 2 mirrors to 5, col 3 mirrors to 4
        const isHair = (r === 1) || (c === 2 && (h[3] & (1 << r)) !== 0);
        grid[r][c] = isHair ? 1 : 2;
        grid[r][7 - c] = grid[r][c];
      }
    }

    // Eyes at row 2, col 3 & 4
    grid[2][3] = 3;
    grid[2][4] = 3;

    // Body / Armor (Rows 4 to 6)
    for (let r = 4; r <= 6; r++) {
      for (let c = 2; c <= 3; c++) {
        grid[r][c] = 4;
        grid[r][7 - c] = 4;
      }
    }

    // Legs / Boots at row 7
    grid[7][2] = 4;
    grid[7][5] = 4;

    // Level-gated equipment overlays:
    if (level >= 4) {
      // Bronze / Iron Pauldrons (Shoulders)
      grid[4][1] = 5;
      grid[4][6] = 5;
    }
    if (level >= 7) {
      // Knight Helm Visor or Crown Peak
      grid[0][3] = 5;
      grid[0][4] = 5;
    }
    if (level >= 10) {
      // Golden Sword / Staff in hand
      grid[3][0] = 5;
      grid[4][0] = 5;
      grid[5][0] = 5;
    }

    const colorMap: Record<number, string> = {
      0: 'transparent',
      1: hair,
      2: skin,
      3: eyeColor,
      4: armor,
      5: level >= 10 ? '#ffd700' : '#a9a9a9', // Gold for top tier, silver/iron otherwise
    };

    return { grid, colorMap };
  }, [username, level]);

  const hasAura = level >= 10;

  return (
    <div
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: size,
        height: size,
        borderRadius: 8,
        background: hasAura ? 'linear-gradient(135deg, #2b1055, #7597de)' : '#1e293b',
        boxShadow: hasAura ? '0 0 10px #ffd700, 0 0 4px #ffaa00' : '0 2px 4px rgba(0,0,0,0.3)',
        padding: 4,
        boxSizing: 'border-box',
        border: hasAura ? '2px solid #ffd700' : '1px solid #334155',
        flexShrink: 0,
      }}
      title={`${username} (Lvl ${level})`}
    >
      <svg
        viewBox="0 0 8 8"
        width={size - 10}
        height={size - 10}
        style={{
          shapeRendering: 'crispEdges',
          imageRendering: 'pixelated',
        }}
      >
        {spriteData.grid.map((row, r) =>
          row.map((val, c) => {
            if (val === 0) return null;
            return (
              <rect
                key={`${r}-${c}`}
                x={c}
                y={r}
                width={1}
                height={1}
                fill={spriteData.colorMap[val]}
              />
            );
          })
        )}
      </svg>
    </div>
  );
};
