in      vec2      t_uv;
uniform sampler2D uv;
uniform float     fd;    // fade distance
uniform float     alpha; // transparency
out     vec4      ffc;   // final fragment colour

float fade(float distance) {
   float z = gl_FragCoord.z / gl_FragCoord.w / distance;
   z = clamp(z, 0.0, 1.0);
   return 1.0 - z;
}
void main() {
   ffc = texture(uv, t_uv);
   ffc.a = ffc.a*fade(fd)*alpha;
}
