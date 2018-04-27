in      vec2      v_t;     // interpolated textured coordinates.
uniform sampler2D uv;
uniform float     fd;      // fade distance
uniform float     alpha;   // transparency
out     vec4      f_color; // final fragment colour

float fade(float distance) {
   float z = gl_FragCoord.z / gl_FragCoord.w / distance;
   z = clamp(z, 0.0, 1.0);
   return 1.0 - z;
}
void main() {
   f_color = texture(uv, v_t);
   f_color.a = f_color.a*fade(fd)*alpha;
}
