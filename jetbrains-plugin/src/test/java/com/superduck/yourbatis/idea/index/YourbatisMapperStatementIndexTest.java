package com.superduck.yourbatis.idea.index;

import org.junit.Test;

import java.util.Map;

import static com.superduck.yourbatis.idea.index.YourbatisMapperStatementIndex.MAPPER_ROOT;
import static com.superduck.yourbatis.idea.index.YourbatisMapperStatementIndex.key;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

public class YourbatisMapperStatementIndexTest {
    @Test
    public void indexesMapperAndSupportedStatementsWithoutResolvingTheDtd() {
        Map<String, Void> keys = YourbatisMapperStatementIndex.extractKeys("""
                <?xml version="1.0" encoding="UTF-8" ?>
                <!DOCTYPE mapper
                        PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
                        "https://mybatis.org/dtd/mybatis-3-mapper.dtd">
                <mapper namespace="github.com/acme/user.UserMapper">
                    <select id="GetByID">SELECT 1</select>
                    <insert id="Create">INSERT INTO users DEFAULT VALUES</insert>
                    <update id="Update">UPDATE users SET active = true</update>
                    <delete id="Delete">DELETE FROM users</delete>
                    <sql id="columns">id</sql>
                </mapper>
                """);

        assertEquals(5, keys.size());
        assertTrue(keys.containsKey(key("UserMapper", MAPPER_ROOT)));
        assertTrue(keys.containsKey(key("UserMapper", "GetByID")));
        assertTrue(keys.containsKey(key("UserMapper", "Create")));
        assertTrue(keys.containsKey(key("UserMapper", "Update")));
        assertTrue(keys.containsKey(key("UserMapper", "Delete")));
    }

    @Test
    public void ignoresNestedStatementLikeTags() {
        Map<String, Void> keys = YourbatisMapperStatementIndex.extractKeys("""
                <mapper namespace="UserMapper">
                    <sql id="fragment">
                        <select id="Nested">SELECT 1</select>
                    </sql>
                    <select id="TopLevel">SELECT 1</select>
                </mapper>
                """);

        assertTrue(keys.containsKey(key("UserMapper", "TopLevel")));
        assertTrue(!keys.containsKey(key("UserMapper", "Nested")));
    }

    @Test
    public void returnsNoPartialKeysForMalformedXml() {
        Map<String, Void> keys = YourbatisMapperStatementIndex.extractKeys("""
                <mapper namespace="UserMapper">
                    <select id="Get">SELECT 1</select>
                """);

        assertTrue(keys.isEmpty());
    }
}
