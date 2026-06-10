window.addEventListener('DOMContentLoaded', () => {
  const canvas = document.getElementById('hacker-canvas');
  if (!canvas) return;

  const ctx = canvas.getContext('2d');

  let width = canvas.width = window.innerWidth;
  let height = canvas.height = window.innerHeight;

  window.addEventListener('resize', () => {
    width = canvas.width = window.innerWidth;
    height = canvas.height = window.innerHeight;
  });

  // Track mouse position
  let mouseX = -1000;
  let mouseY = -1000;
  window.addEventListener('mousemove', (e) => {
    mouseX = e.clientX;
    mouseY = e.clientY;
  });
  window.addEventListener('mouseout', () => {
    mouseX = -1000;
    mouseY = -1000;
  });

  // Hacker Yellow colors
  const colors = [
    'rgba(241, 196, 15, 0.95)',
    'rgba(243, 156, 18, 0.95)',
    'rgba(255, 215, 0, 0.95)',
    'rgba(245, 176, 65, 0.95)',
    'rgba(248, 196, 113, 0.95)'
  ];
  
  const characters = "0101010101010101010101010101SQUEAKCLOUDSHADOWADMINOPSECIAMRBACGCPJSONAUDIT";
  const fontSize = 16;
  let columns = Math.floor(width / fontSize);

  let rainDrops = Array.from({ length: columns }, () => Math.floor(Math.random() * -100));
  let speeds = Array.from({ length: columns }, () => 0.5 + Math.random() * 1.5);

  window.addEventListener('resize', () => {
    const newColumns = Math.floor(width / fontSize);
    if (newColumns > columns) {
      for(let i=columns; i<newColumns; i++){
        rainDrops[i] = Math.floor(Math.random() * -100);
        speeds[i] = 0.5 + Math.random() * 1.5;
      }
    }
    columns = newColumns;
  });

  const draw = () => {
    ctx.fillStyle = 'rgba(5, 5, 5, 0.15)'; // slightly faster fade for trailing effect
    ctx.fillRect(0, 0, width, height);

    ctx.font = 'bold ' + fontSize + 'px monospace';
    ctx.textAlign = 'center';

    for (let i = 0; i < columns; i++) {
      if (rainDrops[i] === undefined) continue;

      const text = characters.charAt(Math.floor(Math.random() * characters.length));
      
      const x = i * fontSize + fontSize/2;
      const y = rainDrops[i] * fontSize;

      // Mouse repulsion calculation
      const dx = mouseX - x;
      const dy = mouseY - y;
      const distance = Math.sqrt(dx * dx + dy * dy);
      
      let offsetX = x;
      let offsetY = y;
      
      if (distance < 120) {
        // Repel the character away from mouse
        const force = (120 - distance) / 120;
        offsetX -= dx * force * 0.5;
        offsetY -= dy * force * 0.5;
        ctx.fillStyle = '#fff'; // Glow bright white when repelled
        ctx.shadowBlur = 15;
        ctx.shadowColor = '#f1c40f';
      } else {
        ctx.fillStyle = colors[Math.floor(Math.random() * colors.length)];
        ctx.shadowBlur = 0;
      }
      
      ctx.fillText(text, offsetX, offsetY);

      if (rainDrops[i] * fontSize > height && Math.random() > 0.95) {
        rainDrops[i] = 0;
        speeds[i] = 0.5 + Math.random() * 1.5;
      }
      rainDrops[i] += speeds[i];
    }
  };

  // 60fps roughly
  setInterval(draw, 30);
});
