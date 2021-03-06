in      vec2      v_t;     // interpolated textured coordinates.
uniform sampler2D uv0;
uniform sampler2D uv1;
uniform float     time;
uniform float     fd;      // fade distance
out     vec4      f_color;

// sping calculates rotated uv coordinates.
vec2 spin(vec2 coords, float now, float rotSpeed) {
   float sa = sin(now*rotSpeed);
   float ca = cos(now*rotSpeed);
   mat2 rot = mat2(ca, -sa, sa, ca);
   return ((coords-0.5)*rot)+0.5;
}

// fade out with distance.
float fade(float distance) {
   float z = gl_FragCoord.z / gl_FragCoord.w / distance;
   z = clamp(z, 0.0, 1.0);
   return 1.0 - z;
}

void main() {
   vec4 t0 = texture(uv0, spin(v_t, time, 1));
   vec4 t1 = texture(uv0, spin(v_t, time, -0.75));
   vec4 t2 = texture(uv1, spin(v_t, time, 1.5));
   vec4 t3 = texture(uv1, spin(v_t, time, -2));
   f_color = mix(mix(t0, t1, 0.5), mix(t2, t3, 0.5), 0.5);
   f_color.a = f_color.a*fade(fd);
}

